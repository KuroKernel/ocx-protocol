import React, { useState } from 'react';
import { Code, Copy, Check, ArrowRight, ExternalLink } from 'lucide-react';

const APIReference = () => {
  const [copiedCode, setCopiedCode] = useState(null);

  const copyToClipboard = (code, id) => {
    navigator.clipboard.writeText(code);
    setCopiedCode(id);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  const endpoints = [
    {
      method: 'POST',
      path: '/api/v1/execute',
      description: 'Execute code with idempotency and generate cryptographic receipt',
      parameters: [
        { name: 'artifact', type: 'string', required: true, description: 'Base64-encoded executable code' },
        { name: 'input', type: 'string', required: true, description: 'Base64-encoded input data' },
        { name: 'max_cycles', type: 'integer', required: false, description: 'Maximum cycles to execute (default: 10000)' }
      ],
      response: {
        receipt_blob: 'string',
        cycles_used: 'integer',
        price_micro_units: 'integer'
      },
      example: {
        request: `curl -X POST http://localhost:8080/api/v1/execute \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: demo-123" \\
  -d '{
    "artifact": "aGVsbG93b3JsZA==",
    "input": "dGVzdGRhdGE=",
    "max_cycles": 1000
  }'`,
        response: `{
  "receipt_blob": "eyJ2ZXJzaW9uIjoi...",
  "cycles_used": 423,
  "price_micro_units": 4230
}`
      }
    },
    {
      method: 'POST',
      path: '/api/v1/verify',
      description: 'Verify cryptographic receipt offline',
      parameters: [
        { name: 'receipt_blob', type: 'string', required: true, description: 'Base64-encoded receipt to verify' }
      ],
      response: {
        valid: 'boolean',
        reason: 'string'
      },
      example: {
        request: `curl -X POST http://localhost:8080/api/v1/verify \\
  -H "Content-Type: application/json" \\
  -d '{
    "receipt_blob": "eyJ2ZXJzaW9uIjoi..."
  }'`,
        response: `{
  "valid": true,
  "reason": "Signature verified"
}`
      }
    },
    {
      method: 'GET',
      path: '/api/v1/receipts',
      description: 'List all receipts with pagination',
      parameters: [
        { name: 'page', type: 'integer', required: false, description: 'Page number (default: 1)' },
        { name: 'limit', type: 'integer', required: false, description: 'Items per page (default: 50)' }
      ],
      response: {
        receipts: 'array',
        total: 'integer',
        page: 'integer',
        limit: 'integer'
      },
      example: {
        request: `curl -X GET "http://localhost:8080/api/v1/receipts?page=1&limit=10"`,
        response: `{
  "receipts": [
    {
      "receipt_hash": "2c26b46b68b68f86...",
      "cycles_used": 423,
      "created_at": "2025-09-20T15:30:45Z"
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 10
}`
      }
    },
    {
      method: 'GET',
      path: '/health',
      description: 'Health check endpoint',
      parameters: [],
      response: {
        status: 'string',
        timestamp: 'string',
        version: 'string'
      },
      example: {
        request: `curl -X GET http://localhost:8080/health`,
        response: `{
  "status": "healthy",
  "timestamp": "2025-09-20T15:30:45Z",
  "version": "v1.0.0-rc.1-pilot1"
}`
      }
    },
    {
      method: 'GET',
      path: '/metrics',
      description: 'Prometheus metrics endpoint',
      parameters: [],
      response: 'Prometheus format metrics',
      example: {
        request: `curl -X GET http://localhost:8080/metrics`,
        response: `# HELP ocx_execute_total Total number of execute requests
# TYPE ocx_execute_total counter
ocx_execute_total 150

# HELP ocx_verify_latency_seconds Latency of verify requests
# TYPE ocx_verify_latency_seconds histogram
ocx_verify_latency_seconds_bucket{le="0.005"} 45
ocx_verify_latency_seconds_bucket{le="0.01"} 89
ocx_verify_latency_seconds_bucket{le="0.02"} 142
ocx_verify_latency_seconds_bucket{le="+Inf"} 150`
      }
    }
  ];

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            API Reference
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Complete API documentation for OCX Protocol. All endpoints, parameters, 
            and response formats.
          </p>
        </div>

        {/* Quick Links */}
        <div className="grid md:grid-cols-3 gap-8 mb-20">
          <a 
            href="/api/openapi.yaml" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Code className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">OpenAPI Spec</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Download the complete OpenAPI 3.0 specification
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>Download YAML</span>
              <ExternalLink className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="#error-codes" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Code className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Error Codes</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Complete reference of all error codes and meanings
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>View errors</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="#rate-limiting" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Code className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Rate Limiting</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Understanding rate limits and best practices
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>Learn more</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>
        </div>

        {/* Endpoints */}
        <div className="space-y-16 mb-20">
          {endpoints.map((endpoint, index) => (
            <div key={index} className="border border-gray-200 rounded-sm">
              <div className="p-8 border-b border-gray-200">
                <div className="flex items-center space-x-4 mb-4">
                  <span className={`px-3 py-1 rounded text-sm font-medium ${
                    endpoint.method === 'POST' ? 'bg-green-100 text-green-800' :
                    endpoint.method === 'GET' ? 'bg-blue-100 text-blue-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {endpoint.method}
                  </span>
                  <code className="text-lg font-mono text-black">{endpoint.path}</code>
                </div>
                <p className="text-gray-600 text-lg">{endpoint.description}</p>
              </div>

              <div className="p-8">
                <div className="grid lg:grid-cols-2 gap-12">
                  {/* Parameters */}
                  <div>
                    <h3 className="text-xl font-medium text-black mb-6">Parameters</h3>
                    {endpoint.parameters.length > 0 ? (
                      <div className="space-y-4">
                        {endpoint.parameters.map((param, paramIndex) => (
                          <div key={paramIndex} className="border-l-4 border-gray-200 pl-4">
                            <div className="flex items-center space-x-2 mb-2">
                              <code className="text-sm font-mono text-black">{param.name}</code>
                              <span className="text-sm text-gray-500">({param.type})</span>
                              {param.required && (
                                <span className="text-xs bg-red-100 text-red-800 px-2 py-1 rounded">
                                  required
                                </span>
                              )}
                            </div>
                            <p className="text-gray-600 text-sm">{param.description}</p>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-gray-500 italic">No parameters required</p>
                    )}
                  </div>

                  {/* Response */}
                  <div>
                    <h3 className="text-xl font-medium text-black mb-6">Response</h3>
                    <div className="bg-gray-50 p-4 rounded-sm">
                      <pre className="text-sm font-mono text-gray-800">
                        {JSON.stringify(endpoint.response, null, 2)}
                      </pre>
                    </div>
                  </div>
                </div>

                {/* Example */}
                <div className="mt-8">
                  <h3 className="text-xl font-medium text-black mb-6">Example</h3>
                  <div className="grid lg:grid-cols-2 gap-8">
                    <div>
                      <h4 className="text-lg font-medium text-black mb-4">Request</h4>
                      <div className="relative">
                        <button
                          onClick={() => copyToClipboard(endpoint.example.request, `request-${index}`)}
                          className="absolute top-2 right-2 p-2 hover:bg-gray-100 rounded"
                        >
                          {copiedCode === `request-${index}` ? (
                            <Check className="w-4 h-4 text-green-600" />
                          ) : (
                            <Copy className="w-4 h-4 text-gray-400" />
                          )}
                        </button>
                        <pre className="bg-black text-green-400 p-4 rounded-sm text-sm overflow-x-auto">
                          {endpoint.example.request}
                        </pre>
                      </div>
                    </div>
                    <div>
                      <h4 className="text-lg font-medium text-black mb-4">Response</h4>
                      <div className="relative">
                        <button
                          onClick={() => copyToClipboard(endpoint.example.response, `response-${index}`)}
                          className="absolute top-2 right-2 p-2 hover:bg-gray-100 rounded"
                        >
                          {copiedCode === `response-${index}` ? (
                            <Check className="w-4 h-4 text-green-600" />
                          ) : (
                            <Copy className="w-4 h-4 text-gray-400" />
                          )}
                        </button>
                        <pre className="bg-gray-900 text-gray-100 p-4 rounded-sm text-sm overflow-x-auto">
                          {endpoint.example.response}
                        </pre>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Error Codes */}
        <div id="error-codes" className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Error Codes</h2>
          
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-4 px-6 font-medium text-black">Code</th>
                  <th className="text-left py-4 px-6 font-medium text-black">HTTP Status</th>
                  <th className="text-left py-4 px-6 font-medium text-black">Description</th>
                </tr>
              </thead>
              <tbody className="text-sm">
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E001</td>
                  <td className="py-4 px-6">400 Bad Request</td>
                  <td className="py-4 px-6 text-gray-600">Missing required field or invalid input</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E002</td>
                  <td className="py-4 px-6">400 Bad Request</td>
                  <td className="py-4 px-6 text-gray-600">Invalid receipt format or malformed data</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E003</td>
                  <td className="py-4 px-6">429 Too Many Requests</td>
                  <td className="py-4 px-6 text-gray-600">Rate limit exceeded</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E004</td>
                  <td className="py-4 px-6">500 Internal Server Error</td>
                  <td className="py-4 px-6 text-gray-600">Execution failed or internal error</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E005</td>
                  <td className="py-4 px-6">500 Internal Server Error</td>
                  <td className="py-4 px-6 text-gray-600">Storage failed or database error</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E006</td>
                  <td className="py-4 px-6">500 Internal Server Error</td>
                  <td className="py-4 px-6 text-gray-600">Internal server error</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 font-mono text-black">E007</td>
                  <td className="py-4 px-6">409 Conflict</td>
                  <td className="py-4 px-6 text-gray-600">Idempotency key mismatch</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Rate Limiting */}
        <div id="rate-limiting" className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Rate Limiting</h2>
          
          <div className="grid lg:grid-cols-2 gap-16">
            <div>
              <h3 className="text-2xl font-medium text-black mb-6">Limits</h3>
              <ul className="space-y-4">
                <li className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
                  <div>
                    <span className="font-medium text-black">Default Rate Limit:</span>
                    <span className="text-gray-600 ml-2">100 requests per second per client</span>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
                  <div>
                    <span className="font-medium text-black">Burst Allowance:</span>
                    <span className="text-gray-600 ml-2">200 requests in 1 second</span>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
                  <div>
                    <span className="font-medium text-black">Body Size Limit:</span>
                    <span className="text-gray-600 ml-2">1MB maximum request body</span>
                  </div>
                </li>
                <li className="flex items-start space-x-3">
                  <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
                  <div>
                    <span className="font-medium text-black">Payload Limits:</span>
                    <span className="text-gray-600 ml-2">10KB for artifact and input fields</span>
                  </div>
                </li>
              </ul>
            </div>
            
            <div>
              <h3 className="text-2xl font-medium text-black mb-6">Headers</h3>
              <div className="space-y-4">
                <div className="bg-gray-50 p-4 rounded-sm">
                  <div className="text-sm font-mono text-black mb-2">X-RateLimit-Limit</div>
                  <div className="text-gray-600 text-sm">Maximum requests per second</div>
                </div>
                <div className="bg-gray-50 p-4 rounded-sm">
                  <div className="text-sm font-mono text-black mb-2">X-RateLimit-Remaining</div>
                  <div className="text-gray-600 text-sm">Remaining requests in current window</div>
                </div>
                <div className="bg-gray-50 p-4 rounded-sm">
                  <div className="text-sm font-mono text-black mb-2">X-RateLimit-Reset</div>
                  <div className="text-gray-600 text-sm">Time when rate limit resets (Unix timestamp)</div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* CTA */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Ready to integrate?
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Download our SDKs, check out examples, or start building with the API.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <a 
              href="#documentation" 
              className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center"
            >
              View Documentation
              <ArrowRight className="w-5 h-5 ml-3" />
            </a>
            <a 
              href="/api/openapi.yaml" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              Download OpenAPI Spec
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default APIReference;
