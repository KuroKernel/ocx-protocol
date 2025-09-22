import React from 'react';
import { ArrowRight, Check, Hash, Shield, Zap, Globe } from 'lucide-react';

const Specification = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            OCX Protocol Specification
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Complete technical specification for the OCX Protocol v1.0.0-rc.1-pilot1. 
            Mathematical proof for computational integrity.
          </p>
        </div>

        {/* Cryptographic Receipt Example */}
        <section className="mb-20">
          <div className="bg-black text-green-400 p-10 rounded-sm font-mono text-base relative max-w-4xl mx-auto">
            <div className="absolute -top-2 -right-2 w-4 h-4 bg-gray-200 rounded-full"></div>
            <div className="absolute -bottom-2 -left-2 w-3 h-3 bg-gray-400 rounded"></div>
            <div className="space-y-3">
              <div className="text-gray-500">// Cryptographic Receipt</div>
              <div>version: "v1-min"</div>
              <div>artifact_hash: "2c26b46b68b68f86..."</div>
              <div>input_hash: "48e80c4b8b405e38..."</div>
              <div>output_hash: "0ff558246733602c..."</div>
              <div>cycles: 423</div>
              <div>transcript_root: "7c8199b723e2dab5..."</div>
              <div>issuer_pubkey: "ed25519:6d4f5dc7..."</div>
              <div>signature: "ed25519:3a6dd14e..."</div>
              <div className="pt-6 text-green-400 flex items-center">
                <div className="w-3 h-3 bg-green-400 rounded-full mr-3"></div>
                VERIFIED
              </div>
            </div>
          </div>
        </section>

        {/* Core Concepts */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Core Concepts</h2>
          
          <div className="grid lg:grid-cols-2 gap-16">
            <div className="space-y-8">
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <Hash className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-4">Deterministic Execution</h3>
                  <p className="text-gray-600 leading-relaxed">
                    Isolated runtime environment ensures identical results across all architectures. 
                    No syscalls, no floating point, no non-determinism. Every execution produces 
                    the same output for the same input.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <Shield className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-4">Cryptographic Receipts</h3>
                  <p className="text-gray-600 leading-relaxed">
                    SHA-256 hashing with Ed25519 signatures. CBOR encoding for universal 
                    compatibility. Immutable proof of execution that can be verified offline.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <Zap className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-4">Cycle Metering</h3>
                  <p className="text-gray-600 leading-relaxed">
                    Precise computational resource measurement. Every instruction cycle 
                    is counted and recorded in the receipt for accurate pricing and 
                    resource management.
                  </p>
                </div>
              </div>
            </div>
            
            <div className="bg-gray-50 p-8 rounded-sm">
              <h4 className="text-lg font-medium text-black mb-6">Receipt Structure</h4>
              <div className="space-y-4 text-sm font-mono">
                <div className="flex justify-between">
                  <span className="text-gray-600">version:</span>
                  <span className="text-black">"v1-min"</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">artifact_hash:</span>
                  <span className="text-black">"2c26b46b68b68f86..."</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">input_hash:</span>
                  <span className="text-black">"48e80c4b8b405e38..."</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">output_hash:</span>
                  <span className="text-black">"0ff558246733602c..."</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">cycles:</span>
                  <span className="text-black">423</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">transcript_root:</span>
                  <span className="text-black">"7c8199b723e2dab5..."</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">issuer_pubkey:</span>
                  <span className="text-black">"ed25519:6d4f5dc7..."</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">signature:</span>
                  <span className="text-black">"ed25519:3a6dd14e..."</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* API Endpoints */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">API Endpoints</h2>
          
          <div className="space-y-8">
            <div className="border border-gray-200 rounded-sm p-6">
              <div className="flex items-center space-x-4 mb-4">
                <span className="bg-green-100 text-green-800 px-3 py-1 rounded text-sm font-medium">POST</span>
                <code className="text-lg font-mono">/api/v1/execute</code>
              </div>
              <p className="text-gray-600 mb-4">Execute code with idempotency and generate cryptographic receipt</p>
              <div className="bg-gray-50 p-4 rounded text-sm font-mono">
                <div>Request: {`{ artifact, input, max_cycles }`}</div>
                <div>Response: {`{ receipt_blob, cycles_used, price_micro_units }`}</div>
              </div>
            </div>
            
            <div className="border border-gray-200 rounded-sm p-6">
              <div className="flex items-center space-x-4 mb-4">
                <span className="bg-blue-100 text-blue-800 px-3 py-1 rounded text-sm font-medium">POST</span>
                <code className="text-lg font-mono">/api/v1/verify</code>
              </div>
              <p className="text-gray-600 mb-4">Verify cryptographic receipt offline</p>
              <div className="bg-gray-50 p-4 rounded text-sm font-mono">
                <div>Request: {`{ receipt_blob }`}</div>
                <div>Response: {`{ valid, reason }`}</div>
              </div>
            </div>
            
            <div className="border border-gray-200 rounded-sm p-6">
              <div className="flex items-center space-x-4 mb-4">
                <span className="bg-gray-100 text-gray-800 px-3 py-1 rounded text-sm font-medium">GET</span>
                <code className="text-lg font-mono">/api/v1/receipts</code>
              </div>
              <p className="text-gray-600 mb-4">List all receipts with pagination</p>
              <div className="bg-gray-50 p-4 rounded text-sm font-mono">
                <div>Query: {`?page=1&limit=50`}</div>
                <div>Response: {`{ receipts: [], total, page, limit }`}</div>
              </div>
            </div>
          </div>
        </section>

        {/* Technical Standards */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Technical Standards</h2>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            <div className="text-center">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 flex items-center justify-center">
                <span className="text-white font-bold">RFC</span>
              </div>
              <h3 className="font-medium text-black mb-3">RFC 7049</h3>
              <p className="text-gray-600">CBOR encoding with canonical serialization for deterministic receipts</p>
            </div>
            
            <div className="text-center">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 flex items-center justify-center">
                <span className="text-white font-bold">RFC</span>
              </div>
              <h3 className="font-medium text-black mb-3">RFC 8032</h3>
              <p className="text-gray-600">Ed25519 cryptographic signatures for receipt authenticity</p>
            </div>
            
            <div className="text-center">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 flex items-center justify-center">
                <Globe className="w-8 h-8 text-white" />
              </div>
              <h3 className="font-medium text-black mb-3">Cross-Platform</h3>
              <p className="text-gray-600">Identical execution on x86_64 and ARM64 architectures</p>
            </div>
            
            <div className="text-center">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 flex items-center justify-center">
                <Zap className="w-8 h-8 text-white" />
              </div>
              <h3 className="font-medium text-black mb-3">Cycle Metering</h3>
              <p className="text-gray-600">Precise computational resource measurement and pricing</p>
            </div>
          </div>
        </section>

        {/* Security Model */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Security Model</h2>
          
          <div className="grid lg:grid-cols-2 gap-16">
            <div>
              <h3 className="text-2xl font-medium text-black mb-8">Cryptographic Security</h3>
              <ul className="space-y-4">
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Ed25519 Signatures</span>
                    <p className="text-gray-600">128-bit equivalent security with fast verification</p>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Domain Separation</span>
                    <p className="text-gray-600">Prevents signature reuse across different contexts</p>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Constant-Time Operations</span>
                    <p className="text-gray-600">Prevents timing attacks on cryptographic operations</p>
                  </div>
                </li>
              </ul>
            </div>
            
            <div>
              <h3 className="text-2xl font-medium text-black mb-8">Input Validation</h3>
              <ul className="space-y-4">
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Size Limits</span>
                    <p className="text-gray-600">1MB request body, 10KB artifact/input fields</p>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Time Limits</span>
                    <p className="text-gray-600">3s header, 5s read, 15s write, 60s idle timeouts</p>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <Check className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                  <div>
                    <span className="font-medium text-black">Rate Limiting</span>
                    <p className="text-gray-600">100 RPS per client with burst allowance</p>
                  </div>
                </li>
              </ul>
            </div>
          </div>
        </section>

        {/* Performance Guarantees */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Performance Guarantees</h2>
          
          <div className="bg-gray-50 p-8 rounded-sm">
            <div className="grid md:grid-cols-3 gap-8 text-center">
              <div>
                <div className="text-3xl font-light text-black mb-2">P99 &lt; 20ms</div>
                <p className="text-gray-600">Verify endpoint latency</p>
              </div>
              <div>
                <div className="text-3xl font-light text-black mb-2">200+ RPS</div>
                <p className="text-gray-600">Per node throughput</p>
              </div>
              <div>
                <div className="text-3xl font-light text-black mb-2">99.9%</div>
                <p className="text-gray-600">Uptime availability</p>
              </div>
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="text-center">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Ready to implement?
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Download the complete specification and start building with OCX Protocol.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <a 
              href="/api/openapi.yaml" 
              className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center"
            >
              Download OpenAPI Spec
              <ArrowRight className="w-5 h-5 ml-3" />
            </a>
            <a 
              href="#documentation" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              View Documentation
            </a>
          </div>
        </section>
      </div>
    </div>
  );
};

export default Specification;
