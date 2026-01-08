// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "forge-std/Test.sol";
import "../src/OCXVerifier.sol";

contract OCXVerifierTest is Test {
    OCXVerifier public verifier;
    OCXReceiptRegistry public registry;

    address public owner = address(1);
    address public user = address(2);

    bytes32 public constant PROGRAM_HASH = keccak256("test-program-v1");
    bytes32 public constant INPUT_HASH = keccak256("test-input");
    bytes32 public constant OUTPUT_HASH = keccak256("test-output");

    function setUp() public {
        vm.startPrank(owner);
        verifier = new OCXVerifier();
        registry = new OCXReceiptRegistry();
        vm.stopPrank();
    }

    // ============================================================================
    // OCXVerifier Tests
    // ============================================================================

    function test_RegisterMerkleRoot() public {
        bytes32 root = keccak256("merkle-root");

        vm.prank(user);
        verifier.registerMerkleRoot(root, 100, "batch-001");

        OCXVerifier.MerkleRoot memory mr = verifier.getMerkleRoot(root);
        assertTrue(mr.registered);
        assertEq(mr.leafCount, 100);
        assertEq(mr.registrar, user);
        assertEq(mr.metadata, "batch-001");
    }

    function test_ComputeReceiptHash() public view {
        OCXVerifier.ReceiptData memory data = OCXVerifier.ReceiptData({
            programHash: PROGRAM_HASH,
            inputHash: INPUT_HASH,
            outputHash: OUTPUT_HASH,
            gasUsed: 1000,
            timestamp: 1704067200,
            issuerId: "test-issuer",
            floatMode: 0
        });

        bytes32 hash = verifier.computeReceiptHash(data);
        assertTrue(hash != bytes32(0));

        // Same input should give same hash
        bytes32 hash2 = verifier.computeReceiptHash(data);
        assertEq(hash, hash2);
    }

    function test_AddTrustedSigner() public {
        bytes32 signerHash = keccak256("signer-pubkey");

        vm.prank(owner);
        verifier.addTrustedSigner(signerHash);

        assertTrue(verifier.trustedSigners(signerHash));
    }

    function test_AddTrustedSigner_Unauthorized() public {
        bytes32 signerHash = keccak256("signer-pubkey");

        vm.prank(user);
        vm.expectRevert(OCXVerifier.Unauthorized.selector);
        verifier.addTrustedSigner(signerHash);
    }

    function test_RemoveTrustedSigner() public {
        bytes32 signerHash = keccak256("signer-pubkey");

        vm.startPrank(owner);
        verifier.addTrustedSigner(signerHash);
        assertTrue(verifier.trustedSigners(signerHash));

        verifier.removeTrustedSigner(signerHash);
        assertFalse(verifier.trustedSigners(signerHash));
        vm.stopPrank();
    }

    function test_TransferOwnership() public {
        vm.prank(owner);
        verifier.transferOwnership(user);

        assertEq(verifier.owner(), user);
    }

    function test_VerifyMerkleProof_Simple() public {
        // Create a simple 2-leaf tree
        bytes32 leaf1 = keccak256("leaf1");
        bytes32 leaf2 = keccak256("leaf2");

        // Root = hash(min(leaf1, leaf2) || max(leaf1, leaf2))
        bytes32 root;
        if (leaf1 <= leaf2) {
            root = keccak256(abi.encodePacked(leaf1, leaf2));
        } else {
            root = keccak256(abi.encodePacked(leaf2, leaf1));
        }

        // Register root
        vm.prank(user);
        verifier.registerMerkleRoot(root, 2, "test");

        // Verify leaf1 with proof [leaf2]
        bytes32[] memory proof = new bytes32[](1);
        proof[0] = leaf2;

        bool valid = verifier.verifyReceiptInTree(leaf1, root, proof, 0);
        assertTrue(valid);
    }

    function test_VerifyMerkleProof_Invalid() public {
        bytes32 leaf1 = keccak256("leaf1");
        bytes32 leaf2 = keccak256("leaf2");
        bytes32 fakeLeaf = keccak256("fake");

        bytes32 root;
        if (leaf1 <= leaf2) {
            root = keccak256(abi.encodePacked(leaf1, leaf2));
        } else {
            root = keccak256(abi.encodePacked(leaf2, leaf1));
        }

        vm.prank(user);
        verifier.registerMerkleRoot(root, 2, "test");

        bytes32[] memory proof = new bytes32[](1);
        proof[0] = leaf2;

        // Should fail with fake leaf
        bool valid = verifier.verifyReceiptInTree(fakeLeaf, root, proof, 0);
        assertFalse(valid);
    }

    function test_VerifyAndRecordReceipt() public {
        // Build a receipt and tree
        OCXVerifier.ReceiptData memory data = OCXVerifier.ReceiptData({
            programHash: PROGRAM_HASH,
            inputHash: INPUT_HASH,
            outputHash: OUTPUT_HASH,
            gasUsed: 1000,
            timestamp: 1704067200,
            issuerId: "test",
            floatMode: 0
        });

        bytes32 receiptHash = verifier.computeReceiptHash(data);
        bytes32 otherLeaf = keccak256("other");

        bytes32 root;
        if (receiptHash <= otherLeaf) {
            root = keccak256(abi.encodePacked(receiptHash, otherLeaf));
        } else {
            root = keccak256(abi.encodePacked(otherLeaf, receiptHash));
        }

        vm.prank(user);
        verifier.registerMerkleRoot(root, 2, "batch");

        bytes32[] memory proof = new bytes32[](1);
        proof[0] = otherLeaf;

        vm.prank(user);
        verifier.verifyAndRecordReceipt(data, root, proof, 0);

        // Check receipt is recorded
        assertTrue(verifier.isReceiptVerified(receiptHash));

        OCXVerifier.Receipt memory receipt = verifier.getReceipt(receiptHash);
        assertEq(receipt.programHash, PROGRAM_HASH);
        assertEq(receipt.inputHash, INPUT_HASH);
        assertEq(receipt.outputHash, OUTPUT_HASH);
        assertEq(receipt.gasUsed, 1000);
        assertEq(receipt.verifier, user);
    }

    function test_VerifyAndRecordReceipt_AlreadyVerified() public {
        OCXVerifier.ReceiptData memory data = OCXVerifier.ReceiptData({
            programHash: PROGRAM_HASH,
            inputHash: INPUT_HASH,
            outputHash: OUTPUT_HASH,
            gasUsed: 1000,
            timestamp: 1704067200,
            issuerId: "test",
            floatMode: 0
        });

        bytes32 receiptHash = verifier.computeReceiptHash(data);
        bytes32 otherLeaf = keccak256("other");

        bytes32 root;
        if (receiptHash <= otherLeaf) {
            root = keccak256(abi.encodePacked(receiptHash, otherLeaf));
        } else {
            root = keccak256(abi.encodePacked(otherLeaf, receiptHash));
        }

        vm.prank(user);
        verifier.registerMerkleRoot(root, 2, "batch");

        bytes32[] memory proof = new bytes32[](1);
        proof[0] = otherLeaf;

        vm.prank(user);
        verifier.verifyAndRecordReceipt(data, root, proof, 0);

        // Try to verify again
        vm.prank(user);
        vm.expectRevert(OCXVerifier.ReceiptAlreadyVerified.selector);
        verifier.verifyAndRecordReceipt(data, root, proof, 0);
    }

    function test_BatchVerify() public {
        // Create 4 leaves
        bytes32[] memory leaves = new bytes32[](4);
        leaves[0] = keccak256("leaf0");
        leaves[1] = keccak256("leaf1");
        leaves[2] = keccak256("leaf2");
        leaves[3] = keccak256("leaf3");

        // Build tree manually
        bytes32 hash01 = _sortedHash(leaves[0], leaves[1]);
        bytes32 hash23 = _sortedHash(leaves[2], leaves[3]);
        bytes32 root = _sortedHash(hash01, hash23);

        vm.prank(user);
        verifier.registerMerkleRoot(root, 4, "batch");

        // Prepare batch verification
        bytes32[] memory receipts = new bytes32[](2);
        receipts[0] = leaves[0];
        receipts[1] = leaves[2];

        bytes32[][] memory proofs = new bytes32[][](2);
        proofs[0] = new bytes32[](2);
        proofs[0][0] = leaves[1];
        proofs[0][1] = hash23;

        proofs[1] = new bytes32[](2);
        proofs[1][0] = leaves[3];
        proofs[1][1] = hash01;

        uint256[] memory indices = new uint256[](2);
        indices[0] = 0;
        indices[1] = 2;

        uint256 validCount = verifier.batchVerify(receipts, root, proofs, indices);
        assertEq(validCount, 2);
    }

    function _sortedHash(bytes32 a, bytes32 b) internal pure returns (bytes32) {
        if (a <= b) {
            return keccak256(abi.encodePacked(a, b));
        }
        return keccak256(abi.encodePacked(b, a));
    }

    // ============================================================================
    // OCXReceiptRegistry Tests
    // ============================================================================

    function test_RegisterProgram() public {
        vm.prank(user);
        registry.registerProgram(PROGRAM_HASH, "Test Program", "1.0.0");

        OCXReceiptRegistry.Program memory prog = registry.getProgram(PROGRAM_HASH);
        assertTrue(prog.registered);
        assertEq(prog.name, "Test Program");
        assertEq(prog.version, "1.0.0");
        assertEq(prog.owner, user);
        assertEq(prog.receiptCount, 0);
    }

    function test_RegisterProgram_AlreadyRegistered() public {
        vm.prank(user);
        registry.registerProgram(PROGRAM_HASH, "Test", "1.0");

        vm.prank(user);
        vm.expectRevert(OCXReceiptRegistry.ProgramAlreadyRegistered.selector);
        registry.registerProgram(PROGRAM_HASH, "Test 2", "2.0");
    }

    function test_RecordReceipt() public {
        vm.prank(user);
        registry.registerProgram(PROGRAM_HASH, "Test", "1.0");

        bytes32 receiptHash = keccak256("receipt-1");

        vm.prank(user);
        registry.recordReceipt(
            PROGRAM_HASH,
            receiptHash,
            INPUT_HASH,
            OUTPUT_HASH,
            1000
        );

        OCXReceiptRegistry.ReceiptRecord memory record = registry.receipts(receiptHash);
        assertEq(record.programHash, PROGRAM_HASH);
        assertEq(record.inputHash, INPUT_HASH);
        assertEq(record.outputHash, OUTPUT_HASH);
        assertEq(record.gasUsed, 1000);

        assertEq(registry.getProgramReceiptCount(PROGRAM_HASH), 1);
    }

    function test_RecordReceipt_ProgramNotRegistered() public {
        bytes32 receiptHash = keccak256("receipt-1");

        vm.prank(user);
        vm.expectRevert(OCXReceiptRegistry.ProgramNotRegistered.selector);
        registry.recordReceipt(
            PROGRAM_HASH,
            receiptHash,
            INPUT_HASH,
            OUTPUT_HASH,
            1000
        );
    }

    function test_GetProgramReceipts_Pagination() public {
        vm.prank(user);
        registry.registerProgram(PROGRAM_HASH, "Test", "1.0");

        // Record 10 receipts
        for (uint256 i = 0; i < 10; i++) {
            bytes32 receiptHash = keccak256(abi.encodePacked("receipt-", i));
            vm.prank(user);
            registry.recordReceipt(PROGRAM_HASH, receiptHash, INPUT_HASH, OUTPUT_HASH, 1000);
        }

        // Get first 5
        bytes32[] memory first5 = registry.getProgramReceipts(PROGRAM_HASH, 0, 5);
        assertEq(first5.length, 5);

        // Get next 5
        bytes32[] memory next5 = registry.getProgramReceipts(PROGRAM_HASH, 5, 5);
        assertEq(next5.length, 5);

        // Get beyond end
        bytes32[] memory beyond = registry.getProgramReceipts(PROGRAM_HASH, 10, 5);
        assertEq(beyond.length, 0);

        // Get partial at end
        bytes32[] memory partial = registry.getProgramReceipts(PROGRAM_HASH, 8, 5);
        assertEq(partial.length, 2);
    }

    function test_TotalCounters() public {
        vm.prank(user);
        registry.registerProgram(PROGRAM_HASH, "Test", "1.0");

        assertEq(registry.totalPrograms(), 1);
        assertEq(registry.totalReceipts(), 0);

        bytes32 receiptHash = keccak256("receipt-1");
        vm.prank(user);
        registry.recordReceipt(PROGRAM_HASH, receiptHash, INPUT_HASH, OUTPUT_HASH, 1000);

        assertEq(registry.totalReceipts(), 1);
    }
}
