import React, { useState } from 'react';
import { ArrowRight, Book, Code, Zap, Shield, Globe, ChevronRight, ChevronDown } from 'lucide-react';

const Documentation = () => {
  const [expandedSection, setExpandedSection] = useState(null);

  const toggleSection = (section) => {
    setExpandedSection(expandedSection === section ? null : section);
  };

  const documentationSections = [
    {
      id: 'getting-started',
      title: 'Getting Started',
      icon: <Zap className="w-6 h-6" />,
      description: 'Quick start guide and basic concepts',
      articles: [
        { title: 'Quick Start', description: 'Get up and running in 5 minutes', href: '#quick-start' },
        { title: 'Core Concepts', description: 'Understanding deterministic execution', href: '#core-concepts' },
        { title: 'Installation', description: 'Install OCX Protocol', href: '#installation' },
        { title: 'First Receipt', description: 'Generate your first cryptographic receipt', href: '#first-receipt' }
      ]
    },
    {
      id: 'api-reference',
      title: 'API Reference',
      icon: <Code className="w-6 h-6" />,
      description: 'Complete API documentation and examples',
      articles: [
        { title: 'Execute Endpoint', description: 'POST /api/v1/execute', href: '#execute-endpoint' },
        { title: 'Verify Endpoint', description: 'POST /api/v1/verify', href: '#verify-endpoint' },
        { title: 'Receipts Endpoint', description: 'GET /api/v1/receipts', href: '#receipts-endpoint' },
        { title: 'Error Codes', description: 'Complete error reference', href: '#error-codes' },
        { title: 'Rate Limiting', description: 'Understanding rate limits', href: '#rate-limiting' }
      ]
    },
    {
      id: 'security',
      title: 'Security',
      icon: <Shield className="w-6 h-6" />,
      description: 'Security model and best practices',
      articles: [
        { title: 'Cryptographic Security', description: 'Ed25519 signatures and hashing', href: '#crypto-security' },
        { title: 'Input Validation', description: 'Size, time, and rate limits', href: '#input-validation' },
        { title: 'Key Management', description: 'Key generation and rotation', href: '#key-management' },
        { title: 'Audit Logging', description: 'Comprehensive audit trails', href: '#audit-logging' }
      ]
    },
    {
      id: 'deployment',
      title: 'Deployment',
      icon: <Globe className="w-6 h-6" />,
      description: 'Production deployment guides',
      articles: [
        { title: 'Docker Deployment', description: 'Containerized deployment', href: '#docker-deployment' },
        { title: 'Kubernetes', description: 'K8s deployment with Helm', href: '#kubernetes' },
        { title: 'Monitoring', description: 'Prometheus and Grafana setup', href: '#monitoring' },
        { title: 'Scaling', description: 'Horizontal and vertical scaling', href: '#scaling' }
      ]
    }
  ];

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            Documentation
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Complete guides, API reference, and examples to help you build 
            with OCX Protocol.
          </p>
        </div>

        {/* Quick Links */}
        <div className="grid md:grid-cols-3 gap-8 mb-20">
          <a 
            href="#quick-start" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Zap className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Quick Start</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Get up and running with OCX Protocol in 5 minutes
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>Get started</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="/api/openapi.yaml" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Code className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">API Reference</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Complete OpenAPI specification and examples
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>View API</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="#examples" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Book className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Examples</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Real-world examples and code samples
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>View examples</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>
        </div>

        {/* Documentation Sections */}
        <div className="space-y-8 mb-20">
          {documentationSections.map((section) => (
            <div key={section.id} className="border border-gray-200 rounded-sm">
              <button
                onClick={() => toggleSection(section.id)}
                className="w-full p-8 text-left hover:bg-gray-50 transition-colors flex items-center justify-between"
              >
                <div className="flex items-center space-x-4">
                  <div className="text-black">{section.icon}</div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-2">{section.title}</h3>
                    <p className="text-gray-600">{section.description}</p>
                  </div>
                </div>
                {expandedSection === section.id ? (
                  <ChevronDown className="w-6 h-6 text-gray-400" />
                ) : (
                  <ChevronRight className="w-6 h-6 text-gray-400" />
                )}
              </button>
              
              {expandedSection === section.id && (
                <div className="border-t border-gray-200 p-8 bg-gray-50">
                  <div className="grid md:grid-cols-2 gap-6">
                    {section.articles.map((article, index) => (
                      <a
                        key={index}
                        href={article.href}
                        className="block p-6 bg-white rounded-sm hover:shadow-sm transition-shadow group"
                      >
                        <h4 className="font-medium text-black mb-2 group-hover:text-gray-600">
                          {article.title}
                        </h4>
                        <p className="text-gray-600 text-sm">{article.description}</p>
                      </a>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>

        {/* Code Examples */}
        <div id="examples" className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Code Examples</h2>
          
          <div className="grid lg:grid-cols-2 gap-12">
            {/* Execute Example */}
            <div>
              <h3 className="text-xl font-medium text-black mb-6">Execute Computation</h3>
              <div className="bg-black text-green-400 p-6 rounded-sm font-mono text-sm overflow-x-auto">
                <div className="text-gray-500"># Execute computation</div>
                <div>curl -X POST http://localhost:8080/api/v1/execute \</div>
                <div className="ml-4">-H "Content-Type: application/json" \</div>
                <div className="ml-4">-H "Idempotency-Key: demo-123" \</div>
                <div className="ml-4">-d '{`{`}'</div>
                <div className="ml-8">"artifact": "aGVsbG93b3JsZA==",</div>
                <div className="ml-8">"input": "dGVzdGRhdGE=",</div>
                <div className="ml-8">"max_cycles": 1000</div>
                <div className="ml-4">{`}`}'</div>
                <div className="mt-4 text-gray-500"># Response</div>
                <div>{`{`}</div>
                <div className="ml-4">"receipt_blob": "eyJ2ZXJzaW9uIjoi...",</div>
                <div className="ml-4">"cycles_used": 423,</div>
                <div className="ml-4">"price_micro_units": 4230</div>
                <div>{`}`}</div>
              </div>
            </div>

            {/* Verify Example */}
            <div>
              <h3 className="text-xl font-medium text-black mb-6">Verify Receipt</h3>
              <div className="bg-black text-green-400 p-6 rounded-sm font-mono text-sm overflow-x-auto">
                <div className="text-gray-500"># Verify receipt</div>
                <div>curl -X POST http://localhost:8080/api/v1/verify \</div>
                <div className="ml-4">-H "Content-Type: application/json" \</div>
                <div className="ml-4">-d '{`{`}'</div>
                <div className="ml-8">"receipt_blob": "eyJ2ZXJzaW9uIjoi..."</div>
                <div className="ml-4">{`}`}'</div>
                <div className="mt-4 text-gray-500"># Response</div>
                <div>{`{`}</div>
                <div className="ml-4">"valid": true,</div>
                <div className="ml-4">"reason": "Signature verified"</div>
                <div>{`}`}</div>
              </div>
            </div>
          </div>
        </div>

        {/* SDKs and Libraries */}
        <div className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">SDKs and Libraries</h2>
          
          <div className="grid md:grid-cols-3 gap-8">
            <div className="border border-gray-200 rounded-sm p-8">
              <h3 className="text-xl font-medium text-black mb-4">JavaScript/Node.js</h3>
              <p className="text-gray-600 mb-6">
                Official SDK for JavaScript and Node.js applications
              </p>
              <div className="space-y-2 text-sm font-mono bg-gray-50 p-4 rounded">
                <div>npm install @ocx-protocol/sdk</div>
                <div>yarn add @ocx-protocol/sdk</div>
              </div>
            </div>

            <div className="border border-gray-200 rounded-sm p-8">
              <h3 className="text-xl font-medium text-black mb-4">Python</h3>
              <p className="text-gray-600 mb-6">
                Python SDK for data science and ML applications
              </p>
              <div className="space-y-2 text-sm font-mono bg-gray-50 p-4 rounded">
                <div>pip install ocx-protocol</div>
                <div>conda install ocx-protocol</div>
              </div>
            </div>

            <div className="border border-gray-200 rounded-sm p-8">
              <h3 className="text-xl font-medium text-black mb-4">Go</h3>
              <p className="text-gray-600 mb-6">
                Go SDK for high-performance applications
              </p>
              <div className="space-y-2 text-sm font-mono bg-gray-50 p-4 rounded">
                <div>go get ocx.local/sdk</div>
                <div>go mod tidy</div>
              </div>
            </div>
          </div>
        </div>

        {/* Community Resources */}
        <div className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12">Community Resources</h2>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            <a 
              href="https://github.com/ocx-protocol/ocx" 
              className="border border-gray-200 rounded-sm p-6 hover:bg-gray-50 transition-colors group"
            >
              <h3 className="font-medium text-black mb-2 group-hover:text-gray-600">GitHub</h3>
              <p className="text-gray-600 text-sm">Source code and issues</p>
            </a>

            <a 
              href="https://discord.gg/ocx-protocol" 
              className="border border-gray-200 rounded-sm p-6 hover:bg-gray-50 transition-colors group"
            >
              <h3 className="font-medium text-black mb-2 group-hover:text-gray-600">Discord</h3>
              <p className="text-gray-600 text-sm">Community chat and support</p>
            </a>

            <a 
              href="https://status.ocx-protocol.com" 
              className="border border-gray-200 rounded-sm p-6 hover:bg-gray-50 transition-colors group"
            >
              <h3 className="font-medium text-black mb-2 group-hover:text-gray-600">Status</h3>
              <p className="text-gray-600 text-sm">Service status and uptime</p>
            </a>

            <a 
              href="https://blog.ocx-protocol.com" 
              className="border border-gray-200 rounded-sm p-6 hover:bg-gray-50 transition-colors group"
            >
              <h3 className="font-medium text-black mb-2 group-hover:text-gray-600">Blog</h3>
              <p className="text-gray-600 text-sm">Updates and tutorials</p>
            </a>
          </div>
        </div>

        {/* CTA */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Need help getting started?
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Join our community or contact our support team for personalized assistance.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <a 
              href="https://discord.gg/ocx-protocol" 
              className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center"
            >
              Join Discord
              <ArrowRight className="w-5 h-5 ml-3" />
            </a>
            <a 
              href="#contact" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              Contact Support
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Documentation;
