// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title OCXVerifier
 * @notice On-chain verifier for OCX Protocol receipts
 * @dev Verifies Merkle proofs and receipt hashes for verifiable computation
 */
contract OCXVerifier {
    // ============================================================================
    // Events
    // ============================================================================

    event ReceiptVerified(
        bytes32 indexed receiptHash,
        bytes32 indexed programHash,
        address indexed verifier,
        uint256 timestamp
    );

    event MerkleRootRegistered(
        bytes32 indexed root,
        uint256 leafCount,
        address indexed registrar,
        uint256 timestamp
    );

    event TrustedSignerAdded(bytes32 indexed publicKeyHash, address indexed addedBy);
    event TrustedSignerRemoved(bytes32 indexed publicKeyHash, address indexed removedBy);

    // ============================================================================
    // Errors
    // ============================================================================

    error InvalidProofLength();
    error InvalidReceiptHash();
    error MerkleRootNotRegistered();
    error MerkleProofFailed();
    error ReceiptAlreadyVerified();
    error SignerNotTrusted();
    error Unauthorized();

    // ============================================================================
    // State
    // ============================================================================

    /// @notice Owner address for administrative functions
    address public owner;

    /// @notice Mapping of registered Merkle roots
    mapping(bytes32 => MerkleRoot) public merkleRoots;

    /// @notice Mapping of verified receipt hashes
    mapping(bytes32 => Receipt) public verifiedReceipts;

    /// @notice Mapping of trusted signer public key hashes
    mapping(bytes32 => bool) public trustedSigners;

    /// @notice Counter for total verified receipts
    uint256 public totalVerifiedReceipts;

    /// @notice Counter for registered Merkle roots
    uint256 public totalMerkleRoots;

    // ============================================================================
    // Structs
    // ============================================================================

    struct MerkleRoot {
        bool registered;
        uint256 leafCount;
        uint256 registeredAt;
        address registrar;
        string metadata;
    }

    struct Receipt {
        bool verified;
        bytes32 programHash;
        bytes32 inputHash;
        bytes32 outputHash;
        uint64 gasUsed;
        uint256 verifiedAt;
        address verifier;
    }

    struct ReceiptData {
        bytes32 programHash;
        bytes32 inputHash;
        bytes32 outputHash;
        uint64 gasUsed;
        uint64 timestamp;
        string issuerId;
        uint8 floatMode; // 0=strict, 1=deterministic, 2=native
    }

    // ============================================================================
    // Constructor
    // ============================================================================

    constructor() {
        owner = msg.sender;
    }

    // ============================================================================
    // Modifiers
    // ============================================================================

    modifier onlyOwner() {
        if (msg.sender != owner) revert Unauthorized();
        _;
    }

    // ============================================================================
    // Admin Functions
    // ============================================================================

    /**
     * @notice Transfer ownership
     * @param newOwner New owner address
     */
    function transferOwnership(address newOwner) external onlyOwner {
        owner = newOwner;
    }

    /**
     * @notice Add a trusted signer (by public key hash)
     * @param publicKeyHash Hash of the Ed25519 public key
     */
    function addTrustedSigner(bytes32 publicKeyHash) external onlyOwner {
        trustedSigners[publicKeyHash] = true;
        emit TrustedSignerAdded(publicKeyHash, msg.sender);
    }

    /**
     * @notice Remove a trusted signer
     * @param publicKeyHash Hash of the Ed25519 public key to remove
     */
    function removeTrustedSigner(bytes32 publicKeyHash) external onlyOwner {
        trustedSigners[publicKeyHash] = false;
        emit TrustedSignerRemoved(publicKeyHash, msg.sender);
    }

    // ============================================================================
    // Merkle Root Registration
    // ============================================================================

    /**
     * @notice Register a Merkle root for a batch of receipts
     * @param root The Merkle root
     * @param leafCount Number of leaves in the tree
     * @param metadata Optional metadata (e.g., batch ID, timestamp range)
     */
    function registerMerkleRoot(
        bytes32 root,
        uint256 leafCount,
        string calldata metadata
    ) external {
        merkleRoots[root] = MerkleRoot({
            registered: true,
            leafCount: leafCount,
            registeredAt: block.timestamp,
            registrar: msg.sender,
            metadata: metadata
        });

        totalMerkleRoots++;

        emit MerkleRootRegistered(root, leafCount, msg.sender, block.timestamp);
    }

    // ============================================================================
    // Verification Functions
    // ============================================================================

    /**
     * @notice Verify a receipt against a registered Merkle root
     * @param receiptHash Hash of the receipt
     * @param root Merkle root
     * @param proof Merkle proof (array of sibling hashes)
     * @param index Index of the receipt in the tree
     * @return valid Whether the proof is valid
     */
    function verifyReceiptInTree(
        bytes32 receiptHash,
        bytes32 root,
        bytes32[] calldata proof,
        uint256 index
    ) external view returns (bool valid) {
        // Check Merkle root is registered
        if (!merkleRoots[root].registered) {
            revert MerkleRootNotRegistered();
        }

        // Verify the Merkle proof
        valid = _verifyMerkleProof(receiptHash, root, proof, index);
    }

    /**
     * @notice Verify and record a receipt
     * @param receiptData The receipt data
     * @param root Merkle root
     * @param proof Merkle proof
     * @param index Index in tree
     */
    function verifyAndRecordReceipt(
        ReceiptData calldata receiptData,
        bytes32 root,
        bytes32[] calldata proof,
        uint256 index
    ) external {
        // Compute receipt hash
        bytes32 receiptHash = computeReceiptHash(receiptData);

        // Check not already verified
        if (verifiedReceipts[receiptHash].verified) {
            revert ReceiptAlreadyVerified();
        }

        // Check Merkle root is registered
        if (!merkleRoots[root].registered) {
            revert MerkleRootNotRegistered();
        }

        // Verify Merkle proof
        if (!_verifyMerkleProof(receiptHash, root, proof, index)) {
            revert MerkleProofFailed();
        }

        // Record the verified receipt
        verifiedReceipts[receiptHash] = Receipt({
            verified: true,
            programHash: receiptData.programHash,
            inputHash: receiptData.inputHash,
            outputHash: receiptData.outputHash,
            gasUsed: receiptData.gasUsed,
            verifiedAt: block.timestamp,
            verifier: msg.sender
        });

        totalVerifiedReceipts++;

        emit ReceiptVerified(
            receiptHash,
            receiptData.programHash,
            msg.sender,
            block.timestamp
        );
    }

    /**
     * @notice Batch verify multiple receipts
     * @param receiptHashes Array of receipt hashes
     * @param root Merkle root
     * @param proofs Array of Merkle proofs
     * @param indices Array of indices
     * @return validCount Number of valid receipts
     */
    function batchVerify(
        bytes32[] calldata receiptHashes,
        bytes32 root,
        bytes32[][] calldata proofs,
        uint256[] calldata indices
    ) external view returns (uint256 validCount) {
        if (!merkleRoots[root].registered) {
            revert MerkleRootNotRegistered();
        }

        for (uint256 i = 0; i < receiptHashes.length; i++) {
            if (_verifyMerkleProof(receiptHashes[i], root, proofs[i], indices[i])) {
                validCount++;
            }
        }
    }

    // ============================================================================
    // View Functions
    // ============================================================================

    /**
     * @notice Check if a receipt has been verified
     * @param receiptHash Hash of the receipt
     * @return verified Whether the receipt is verified
     */
    function isReceiptVerified(bytes32 receiptHash) external view returns (bool) {
        return verifiedReceipts[receiptHash].verified;
    }

    /**
     * @notice Get receipt details
     * @param receiptHash Hash of the receipt
     * @return receipt The receipt details
     */
    function getReceipt(bytes32 receiptHash) external view returns (Receipt memory) {
        return verifiedReceipts[receiptHash];
    }

    /**
     * @notice Check if a Merkle root is registered
     * @param root The Merkle root
     * @return registered Whether the root is registered
     */
    function isMerkleRootRegistered(bytes32 root) external view returns (bool) {
        return merkleRoots[root].registered;
    }

    /**
     * @notice Get Merkle root details
     * @param root The Merkle root
     * @return merkleRoot The Merkle root details
     */
    function getMerkleRoot(bytes32 root) external view returns (MerkleRoot memory) {
        return merkleRoots[root];
    }

    // ============================================================================
    // Hash Functions
    // ============================================================================

    /**
     * @notice Compute the hash of a receipt
     * @param data Receipt data
     * @return hash The receipt hash
     */
    function computeReceiptHash(ReceiptData calldata data) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(
            data.programHash,
            data.inputHash,
            data.outputHash,
            data.gasUsed,
            data.timestamp,
            data.issuerId,
            data.floatMode
        ));
    }

    /**
     * @notice Compute hash for Merkle tree leaf
     * @param data Raw data
     * @return hash The leaf hash
     */
    function computeLeafHash(bytes calldata data) external pure returns (bytes32) {
        return keccak256(data);
    }

    // ============================================================================
    // Internal Functions
    // ============================================================================

    /**
     * @dev Verify a Merkle proof
     * @param leaf Leaf hash
     * @param root Expected root
     * @param proof Array of sibling hashes
     * @param index Leaf index
     * @return valid Whether the proof is valid
     */
    function _verifyMerkleProof(
        bytes32 leaf,
        bytes32 root,
        bytes32[] calldata proof,
        uint256 index
    ) internal pure returns (bool) {
        bytes32 computedHash = leaf;

        for (uint256 i = 0; i < proof.length; i++) {
            bytes32 proofElement = proof[i];

            // Sort hashes for deterministic ordering (same as OCX Protocol)
            if (computedHash <= proofElement) {
                computedHash = keccak256(abi.encodePacked(computedHash, proofElement));
            } else {
                computedHash = keccak256(abi.encodePacked(proofElement, computedHash));
            }

            index = index / 2;
        }

        return computedHash == root;
    }
}

/**
 * @title OCXReceiptRegistry
 * @notice Registry for tracking OCX receipts across multiple programs
 */
contract OCXReceiptRegistry {
    // ============================================================================
    // Events
    // ============================================================================

    event ProgramRegistered(
        bytes32 indexed programHash,
        string name,
        address indexed owner
    );

    event ReceiptRecorded(
        bytes32 indexed programHash,
        bytes32 indexed receiptHash,
        uint256 timestamp
    );

    // ============================================================================
    // Errors
    // ============================================================================

    error ProgramNotRegistered();
    error ProgramAlreadyRegistered();
    error Unauthorized();

    // ============================================================================
    // State
    // ============================================================================

    struct Program {
        bool registered;
        string name;
        string version;
        address owner;
        uint256 receiptCount;
        uint256 registeredAt;
    }

    struct ReceiptRecord {
        bytes32 programHash;
        bytes32 inputHash;
        bytes32 outputHash;
        uint64 gasUsed;
        uint256 recordedAt;
    }

    mapping(bytes32 => Program) public programs;
    mapping(bytes32 => ReceiptRecord) public receipts;
    mapping(bytes32 => bytes32[]) public programReceipts; // programHash => receiptHashes

    uint256 public totalPrograms;
    uint256 public totalReceipts;

    // ============================================================================
    // Program Management
    // ============================================================================

    /**
     * @notice Register a new program
     * @param programHash Hash of the program bytecode
     * @param name Human-readable name
     * @param version Program version
     */
    function registerProgram(
        bytes32 programHash,
        string calldata name,
        string calldata version
    ) external {
        if (programs[programHash].registered) {
            revert ProgramAlreadyRegistered();
        }

        programs[programHash] = Program({
            registered: true,
            name: name,
            version: version,
            owner: msg.sender,
            receiptCount: 0,
            registeredAt: block.timestamp
        });

        totalPrograms++;

        emit ProgramRegistered(programHash, name, msg.sender);
    }

    /**
     * @notice Record a receipt for a program
     * @param programHash Hash of the program
     * @param receiptHash Hash of the receipt
     * @param inputHash Hash of inputs
     * @param outputHash Hash of outputs
     * @param gasUsed Gas consumed
     */
    function recordReceipt(
        bytes32 programHash,
        bytes32 receiptHash,
        bytes32 inputHash,
        bytes32 outputHash,
        uint64 gasUsed
    ) external {
        if (!programs[programHash].registered) {
            revert ProgramNotRegistered();
        }

        receipts[receiptHash] = ReceiptRecord({
            programHash: programHash,
            inputHash: inputHash,
            outputHash: outputHash,
            gasUsed: gasUsed,
            recordedAt: block.timestamp
        });

        programReceipts[programHash].push(receiptHash);
        programs[programHash].receiptCount++;
        totalReceipts++;

        emit ReceiptRecorded(programHash, receiptHash, block.timestamp);
    }

    // ============================================================================
    // View Functions
    // ============================================================================

    /**
     * @notice Get program details
     * @param programHash Hash of the program
     * @return program The program details
     */
    function getProgram(bytes32 programHash) external view returns (Program memory) {
        return programs[programHash];
    }

    /**
     * @notice Get receipt count for a program
     * @param programHash Hash of the program
     * @return count Number of receipts
     */
    function getProgramReceiptCount(bytes32 programHash) external view returns (uint256) {
        return programs[programHash].receiptCount;
    }

    /**
     * @notice Get receipts for a program (paginated)
     * @param programHash Hash of the program
     * @param offset Starting index
     * @param limit Maximum number to return
     * @return receiptHashes Array of receipt hashes
     */
    function getProgramReceipts(
        bytes32 programHash,
        uint256 offset,
        uint256 limit
    ) external view returns (bytes32[] memory) {
        bytes32[] storage allReceipts = programReceipts[programHash];
        uint256 total = allReceipts.length;

        if (offset >= total) {
            return new bytes32[](0);
        }

        uint256 count = limit;
        if (offset + count > total) {
            count = total - offset;
        }

        bytes32[] memory result = new bytes32[](count);
        for (uint256 i = 0; i < count; i++) {
            result[i] = allReceipts[offset + i];
        }

        return result;
    }
}
