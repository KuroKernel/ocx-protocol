import React from 'react';
import { Code, Shield, Zap, Github } from 'lucide-react';

const About = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <h1 className="text-5xl font-light tracking-tight text-black mb-8">
          About OCX
        </h1>

        <div className="space-y-8">
          <p className="text-xl text-gray-700 leading-relaxed">
            OCX Protocol is an open-source system for creating cryptographic proofs
            of program execution. Think of it as a tamper-evident seal for computation.
          </p>

          <div className="border-l-4 border-black pl-6 my-12">
            <p className="text-lg text-gray-700 italic">
              "Same inputs, same code, same results—every time. Provably."
            </p>
          </div>

          <h2 className="text-3xl font-light text-black mt-12 mb-6">What it does</h2>
          <p className="text-gray-700 leading-relaxed">
            When you run a program through OCX, you get a cryptographically signed receipt
            that proves exactly what was executed, with what inputs, and what the results were.
            Anyone can verify this receipt offline without trusting the executor.
          </p>

          <h2 className="text-3xl font-light text-black mt-12 mb-6">How it works</h2>
          <div className="grid md:grid-cols-3 gap-8 my-8">
            <div className="border border-gray-200 p-6 rounded">
              <Code className="w-8 h-8 mb-4 text-black" />
              <h3 className="font-medium text-black mb-2">Deterministic VM</h3>
              <p className="text-sm text-gray-600">Isolated execution environment ensures identical results across all platforms</p>
            </div>
            <div className="border border-gray-200 p-6 rounded">
              <Shield className="w-8 h-8 mb-4 text-black" />
              <h3 className="font-medium text-black mb-2">Cryptographic Receipts</h3>
              <p className="text-sm text-gray-600">Ed25519 signatures with CBOR encoding for universal verification</p>
            </div>
            <div className="border border-gray-200 p-6 rounded">
              <Zap className="w-8 h-8 mb-4 text-black" />
              <h3 className="font-medium text-black mb-2">Offline Verification</h3>
              <p className="text-sm text-gray-600">Verify receipts anywhere without network dependency</p>
            </div>
          </div>

          <h2 className="text-3xl font-light text-black mt-12 mb-6">Use cases</h2>
          <ul className="space-y-3 text-gray-700">
            <li className="flex items-start">
              <span className="text-black mr-3">•</span>
              <span>Prove ML model outputs for regulatory compliance</span>
            </li>
            <li className="flex items-start">
              <span className="text-black mr-3">•</span>
              <span>Create audit trails for financial calculations</span>
            </li>
            <li className="flex items-start">
              <span className="text-black mr-3">•</span>
              <span>Enable reproducible scientific computing</span>
            </li>
            <li className="flex items-start">
              <span className="text-black mr-3">•</span>
              <span>Verify content transformation pipelines</span>
            </li>
          </ul>

          <h2 className="text-3xl font-light text-black mt-12 mb-6">Open source</h2>
          <p className="text-gray-700 leading-relaxed mb-6">
            OCX Protocol is MIT licensed. The protocol specification, implementation,
            and verification tools are all open source.
          </p>

          <a
            href="https://github.com/your-org/ocx-protocol"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center px-6 py-3 bg-black text-white rounded hover:bg-gray-800 transition-colors"
          >
            <Github className="w-5 h-5 mr-2" />
            View on GitHub
          </a>

          <div className="mt-16 p-8 bg-gray-50 rounded">
            <h3 className="text-xl font-medium text-black mb-4">Technical details</h3>
            <div className="grid md:grid-cols-2 gap-4 text-sm text-gray-700">
              <div>
                <span className="font-medium">Signature:</span> Ed25519
              </div>
              <div>
                <span className="font-medium">Encoding:</span> Canonical CBOR
              </div>
              <div>
                <span className="font-medium">Hashing:</span> SHA-256
              </div>
              <div>
                <span className="font-medium">Runtime:</span> Deterministic VM
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default About;
