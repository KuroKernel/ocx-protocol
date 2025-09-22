//! A strict parser for canonical OCX-CBOR v1.1.
//!
//! This parser rejects any non-canonical encoding, ensuring perfect determinism.
//! 
//! Canonical encoding rules:
//! - Minimal encoding for all types
//! - Map keys sorted by canonical order
//! - No indefinite-length items
//! - No duplicate map keys
//! - UTF-8 validation for text strings

use crate::VerificationError;
use std::collections::BTreeMap;
use std::cmp::Ordering;

/// A CBOR value that adheres to canonical encoding rules.
#[derive(Debug, Clone, PartialEq)]
pub enum CanonicalValue {
    /// Unsigned integer (0..=18446744073709551615)
    Integer(u64),
    /// Byte string
    Bytes(Vec<u8>),
    /// UTF-8 text string
    Text(String),
    /// Array of values
    Array(Vec<CanonicalValue>),
    /// Map with canonical key ordering
    Map(BTreeMap<CanonicalValue, CanonicalValue>),
}

impl Eq for CanonicalValue {}

impl Ord for CanonicalValue {
    fn cmp(&self, other: &Self) -> Ordering {
        // Canonical ordering for CBOR values
        match (self, other) {
            (CanonicalValue::Integer(a), CanonicalValue::Integer(b)) => a.cmp(b),
            (CanonicalValue::Bytes(a), CanonicalValue::Bytes(b)) => a.cmp(b),
            (CanonicalValue::Text(a), CanonicalValue::Text(b)) => a.cmp(b),
            (CanonicalValue::Array(a), CanonicalValue::Array(b)) => a.cmp(b),
            (CanonicalValue::Map(a), CanonicalValue::Map(b)) => a.cmp(b),
            // Cross-type ordering: integers < bytes < text < arrays < maps
            (CanonicalValue::Integer(_), _) => Ordering::Less,
            (_, CanonicalValue::Integer(_)) => Ordering::Greater,
            (CanonicalValue::Bytes(_), CanonicalValue::Text(_)) => Ordering::Less,
            (CanonicalValue::Bytes(_), CanonicalValue::Array(_)) => Ordering::Less,
            (CanonicalValue::Bytes(_), CanonicalValue::Map(_)) => Ordering::Less,
            (CanonicalValue::Text(_), CanonicalValue::Bytes(_)) => Ordering::Greater,
            (CanonicalValue::Text(_), CanonicalValue::Array(_)) => Ordering::Less,
            (CanonicalValue::Text(_), CanonicalValue::Map(_)) => Ordering::Less,
            (CanonicalValue::Array(_), CanonicalValue::Bytes(_)) => Ordering::Greater,
            (CanonicalValue::Array(_), CanonicalValue::Text(_)) => Ordering::Greater,
            (CanonicalValue::Array(_), CanonicalValue::Map(_)) => Ordering::Less,
            (CanonicalValue::Map(_), _) => Ordering::Greater,
        }
    }
}

impl PartialOrd for CanonicalValue {
    fn partial_cmp(&self, other: &Self) -> Option<Ordering> {
        Some(self.cmp(other))
    }
}

/// CBOR parser that enforces canonical encoding.
#[derive(Debug)]
pub struct CborParser<'a> {
    data: &'a [u8],
    position: usize,
}

impl<'a> CborParser<'a> {
    /// Create a new parser for the given data.
    pub fn new(data: &'a [u8]) -> Self {
        Self { data, position: 0 }
    }

    /// Parses the entire input as a single CanonicalValue, ensuring no trailing data.
    pub fn parse_full(mut self) -> Result<CanonicalValue, VerificationError> {
        let value = self.parse_value()?;
        if self.position != self.data.len() {
            return Err(VerificationError::NonCanonicalCbor);
        }
        Ok(value)
    }

    fn parse_value(&mut self) -> Result<CanonicalValue, VerificationError> {
        let initial_byte = self.read_byte()?;
        let major_type = initial_byte >> 5;
        let additional_info = initial_byte & 0x1f;

        match major_type {
            0 => self.parse_unsigned_integer(additional_info),
            2 => self.parse_bytes(additional_info),
            3 => self.parse_text(additional_info),
            4 => self.parse_array(additional_info),
            5 => self.parse_map(additional_info),
            _ => Err(VerificationError::InvalidCbor),
        }
    }

    fn parse_unsigned_integer(&mut self, additional_info: u8) -> Result<CanonicalValue, VerificationError> {
        let value = match additional_info {
            0..=23 => additional_info as u64,
            24 => {
                let byte = self.read_byte()? as u64;
                // Must use minimal encoding
                if byte < 24 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                byte
            }
            25 => {
                let bytes = self.read_bytes(2)?;
                let value = u16::from_be_bytes([bytes[0], bytes[1]]) as u64;
                // Must use minimal encoding
                if value < 256 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            26 => {
                let bytes = self.read_bytes(4)?;
                let value = u32::from_be_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]) as u64;
                // Must use minimal encoding
                if value < 65536 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            27 => {
                let bytes = self.read_bytes(8)?;
                let value = u64::from_be_bytes([
                    bytes[0], bytes[1], bytes[2], bytes[3],
                    bytes[4], bytes[5], bytes[6], bytes[7],
                ]);
                // Must use minimal encoding
                if value < 4294967296 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            _ => return Err(VerificationError::InvalidCbor),
        };
        Ok(CanonicalValue::Integer(value))
    }

    fn parse_bytes(&mut self, additional_info: u8) -> Result<CanonicalValue, VerificationError> {
        let length = self.parse_length(additional_info)?;
        let bytes = self.read_bytes(length)?;
        Ok(CanonicalValue::Bytes(bytes.to_vec()))
    }

    fn parse_text(&mut self, additional_info: u8) -> Result<CanonicalValue, VerificationError> {
        let length = self.parse_length(additional_info)?;
        let bytes = self.read_bytes(length)?;
        let text = String::from_utf8(bytes.to_vec())
            .map_err(|_| VerificationError::InvalidUtf8)?;
        Ok(CanonicalValue::Text(text))
    }

    fn parse_array(&mut self, additional_info: u8) -> Result<CanonicalValue, VerificationError> {
        let length = self.parse_length(additional_info)?;
        let mut items = Vec::with_capacity(length);
        for _ in 0..length {
            items.push(self.parse_value()?);
        }
        Ok(CanonicalValue::Array(items))
    }

    fn parse_map(&mut self, additional_info: u8) -> Result<CanonicalValue, VerificationError> {
        let length = self.parse_length(additional_info)?;
        let mut map = BTreeMap::new();
        let mut previous_key: Option<CanonicalValue> = None;

        for _ in 0..length {
            let key = self.parse_value()?;
            let value = self.parse_value()?;

            // Ensure keys are in canonical order
            if let Some(ref prev_key) = previous_key {
                if key <= *prev_key {
                    return Err(VerificationError::NonCanonicalCbor);
                }
            }

            // Check for duplicate keys
            if map.insert(key.clone(), value).is_some() {
                return Err(VerificationError::NonCanonicalCbor);
            }

            previous_key = Some(key);
        }

        Ok(CanonicalValue::Map(map))
    }

    fn parse_length(&mut self, additional_info: u8) -> Result<usize, VerificationError> {
        let length = match additional_info {
            0..=23 => additional_info as u64,
            24 => {
                let byte = self.read_byte()? as u64;
                if byte < 24 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                byte
            }
            25 => {
                let bytes = self.read_bytes(2)?;
                let value = u16::from_be_bytes([bytes[0], bytes[1]]) as u64;
                if value < 256 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            26 => {
                let bytes = self.read_bytes(4)?;
                let value = u32::from_be_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]) as u64;
                if value < 65536 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            27 => {
                let bytes = self.read_bytes(8)?;
                let value = u64::from_be_bytes([
                    bytes[0], bytes[1], bytes[2], bytes[3],
                    bytes[4], bytes[5], bytes[6], bytes[7],
                ]);
                if value < 4294967296 {
                    return Err(VerificationError::NonCanonicalCbor);
                }
                value
            }
            _ => return Err(VerificationError::InvalidCbor),
        };

        // Check for reasonable length limits
        if length > usize::MAX as u64 {
            return Err(VerificationError::IntegerOverflow);
        }

        Ok(length as usize)
    }

    fn read_byte(&mut self) -> Result<u8, VerificationError> {
        if self.position >= self.data.len() {
            return Err(VerificationError::UnexpectedEof);
        }
        let byte = self.data[self.position];
        self.position += 1;
        Ok(byte)
    }

    fn read_bytes(&mut self, count: usize) -> Result<&[u8], VerificationError> {
        if self.position + count > self.data.len() {
            return Err(VerificationError::UnexpectedEof);
        }
        let bytes = &self.data[self.position..self.position + count];
        self.position += count;
        Ok(bytes)
    }
}
