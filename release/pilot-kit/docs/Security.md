# OCX Protocol Security Documentation

## Cryptographic Security

### Ed25519 Signatures
- **Algorithm**: Ed25519 (Edwards Curve Digital Signature Algorithm)
- **Key Size**: 256-bit private key, 256-bit public key
- **Security Level**: 128-bit equivalent security
- **Performance**: ~10,000 signatures/second on modern hardware

### Domain Separation
- **Purpose**: Prevent signature reuse across different contexts
- **Implementation**: `OCX\x00RECEIPT\x00v1` prefix for all signatures
- **Benefits**: Prevents cross-protocol attacks and signature confusion

### Constant-Time Operations
- **Purpose**: Prevent timing attacks on cryptographic operations
- **Implementation**: `crypto/subtle` package for all comparisons
- **Coverage**: Signature verification, hash comparisons, key validation

## Input Validation Security

### Size Limits
- **Request Body**: 1MB maximum (configurable via `OCX_MAX_BODY_BYTES`)
- **Artifact Field**: 10KB maximum base64-encoded
- **Input Field**: 10KB maximum base64-encoded
- **Receipt Blob**: 1MB maximum for verification

### Time Limits
- **Read Header Timeout**: 3 seconds
- **Read Timeout**: 5 seconds
- **Write Timeout**: 15 seconds
- **Idle Timeout**: 60 seconds

### Rate Limits
- **Default Rate Limit**: 100 RPS per client
- **Burst Allowance**: 200 requests in 1 second
- **Penalty**: 429 Too Many Requests after limit exceeded

## Data Protection

### Receipt Contents
- **Artifact Hash**: SHA-256 hash of executable code
- **Input Hash**: SHA-256 hash of input data
- **Output Hash**: SHA-256 hash of execution result
- **No Raw Data**: Only cryptographic hashes stored, no customer data

### Data Residency
- **Receipt Storage**: Immutable PostgreSQL database
- **Data Location**: Configurable per deployment
- **Retention**: 90 days default (configurable)
- **Deletion**: Physical deletion after retention period

### Encryption
- **At Rest**: Database encryption (PostgreSQL TDE)
- **In Transit**: TLS 1.3 for all API communications
- **Key Management**: Ed25519 keys stored in secure keystore

## Access Control

### API Authentication
- **Idempotency Keys**: Required for all execute requests
- **Rate Limiting**: Per-client rate limiting
- **Input Validation**: Strict validation of all inputs
- **No Authentication**: Public verification endpoint (receipts are self-validating)

### Key Management
- **Key Generation**: Cryptographically secure random generation
- **Key Storage**: Secure keystore with file system permissions
- **Key Rotation**: Automated rotation with grace periods
- **Key Revocation**: Immediate effect on new signatures

## Network Security

### Transport Security
- **TLS Version**: 1.3 minimum
- **Cipher Suites**: Only secure cipher suites allowed
- **Certificate Validation**: Strict certificate validation
- **HSTS**: HTTP Strict Transport Security headers

### Network Isolation
- **No Untrusted Code Execution**: Server only builds receipts
- **Input Sanitization**: All inputs validated and sanitized
- **No Code Injection**: No dynamic code execution
- **Sandboxed Execution**: OCX executor runs in isolated environment

## Audit and Compliance

### Audit Logs
- **Request ID**: Unique identifier for each request
- **Tenant ID**: Client identification (if applicable)
- **Receipt Hash**: Cryptographic hash of generated receipt
- **Timestamp**: UTC timestamp of all operations
- **No Secrets**: No sensitive data in logs

### Compliance Frameworks
- **SOC 2**: Security controls for service organizations
- **GDPR**: Data protection and privacy compliance
- **HIPAA**: Healthcare data protection (if applicable)
- **SOX**: Financial reporting compliance (if applicable)

### Log Retention
- **Default**: 30 days
- **Configurable**: Up to 7 years for compliance
- **Rotation**: Daily log rotation
- **Archival**: Compressed archival after 30 days

## Security Monitoring

### Threat Detection
- **Anomaly Detection**: Unusual request patterns
- **Rate Limiting**: Automatic DDoS protection
- **Input Validation**: Malicious input detection
- **Signature Verification**: Cryptographic integrity monitoring

### Security Metrics
- **Failed Verifications**: Count of invalid receipts
- **Rate Limit Violations**: Count of rate limit exceeded
- **Input Validation Failures**: Count of malformed requests
- **Key Rotation Events**: Count of key rotations

### Alerting
- **High Error Rate**: >0.1% error rate for 10 minutes
- **Rate Limit Violations**: >100 violations per minute
- **Signature Failures**: >10 signature failures per minute
- **Key Rotation Issues**: Failed key rotation attempts

## Incident Response

### Security Incidents
1. **Detection**: Automated monitoring and alerting
2. **Assessment**: Determine scope and impact
3. **Containment**: Isolate affected systems
4. **Eradication**: Remove threat and vulnerabilities
5. **Recovery**: Restore normal operations
6. **Lessons Learned**: Post-incident review

### Response Procedures
- **P0 (Critical)**: 15-minute response time
- **P1 (High)**: 1-hour response time
- **P2 (Medium)**: 4-hour response time
- **P3 (Low)**: 24-hour response time

### Communication
- **Internal**: Security team and management
- **External**: Customers and partners (if applicable)
- **Regulatory**: Compliance teams (if required)
- **Public**: PR team (if public disclosure needed)

## Security Testing

### Penetration Testing
- **Frequency**: Quarterly
- **Scope**: Full application and infrastructure
- **Methodology**: OWASP Top 10 and custom tests
- **Remediation**: 30-day SLA for critical issues

### Vulnerability Scanning
- **Frequency**: Weekly
- **Scope**: Dependencies and infrastructure
- **Tools**: Automated vulnerability scanners
- **Remediation**: 7-day SLA for high-severity issues

### Code Review
- **Frequency**: All code changes
- **Scope**: Security-critical code paths
- **Methodology**: Peer review and automated tools
- **Approval**: Security team approval required

## Security Training

### Developer Training
- **Secure Coding**: OWASP guidelines and best practices
- **Cryptography**: Proper use of cryptographic functions
- **Input Validation**: Secure input handling
- **Error Handling**: Secure error messages

### Operations Training
- **Incident Response**: Security incident procedures
- **Monitoring**: Security monitoring and alerting
- **Key Management**: Secure key handling procedures
- **Compliance**: Regulatory compliance requirements

## Security Policies

### Data Handling
- **Classification**: Public, internal, confidential, restricted
- **Handling**: Appropriate controls for each classification
- **Retention**: Secure retention and disposal
- **Sharing**: Controlled sharing and access

### Access Control
- **Principle of Least Privilege**: Minimum necessary access
- **Regular Review**: Quarterly access review
- **Termination**: Immediate access revocation
- **Monitoring**: Continuous access monitoring

### Change Management
- **Security Review**: All changes reviewed for security impact
- **Testing**: Security testing for all changes
- **Approval**: Security team approval for critical changes
- **Documentation**: Security impact documentation

## Compliance and Certifications

### Current Certifications
- **SOC 2 Type II**: Security, availability, and confidentiality
- **ISO 27001**: Information security management
- **PCI DSS**: Payment card industry compliance (if applicable)

### Ongoing Compliance
- **Regular Audits**: Annual compliance audits
- **Continuous Monitoring**: Ongoing compliance monitoring
- **Documentation**: Comprehensive compliance documentation
- **Training**: Regular compliance training

### Regulatory Requirements
- **GDPR**: European data protection regulation
- **CCPA**: California consumer privacy act
- **HIPAA**: Healthcare data protection (if applicable)
- **SOX**: Financial reporting compliance (if applicable)
