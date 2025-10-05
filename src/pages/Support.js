import React from 'react';
import { Mail, Book, Github } from 'lucide-react';

const Support = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <h1 className="text-5xl font-light tracking-tight text-black mb-8">
          Support
        </h1>

        <div className="space-y-12">
          <p className="text-xl text-gray-700 leading-relaxed">
            Need help with OCX Protocol? Here's how to get assistance.
          </p>

          <div className="grid md:grid-cols-2 gap-8">
            <a
              href="mailto:contact@ocx.world"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Mail className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">Email Support</h2>
              <p className="text-gray-600 mb-4">For technical questions and commercial inquiries</p>
              <p className="text-sm text-gray-600">contact@ocx.world</p>
              <p className="text-sm text-gray-500 mt-2">Response time: 1-2 business days</p>
            </a>

            <a
              href="https://github.com/your-org/ocx-protocol/issues"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Github className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">GitHub Issues</h2>
              <p className="text-gray-600 mb-4">For bug reports and feature requests</p>
              <p className="text-sm text-gray-600">Open source repository</p>
              <p className="text-sm text-gray-500 mt-2">Public issue tracking</p>
            </a>
          </div>

          <div className="bg-gray-50 p-8 rounded mt-12">
            <h2 className="text-2xl font-medium text-black mb-6">Common Questions</h2>

            <div className="space-y-6">
              <div>
                <h3 className="font-medium text-black mb-2">How do I get started?</h3>
                <p className="text-gray-700">
                  Check the <a href="https://github.com/your-org/ocx-protocol#quick-start" className="underline">Quick Start</a> in
                  the README. You can run the demo in under 5 minutes.
                </p>
              </div>

              <div>
                <h3 className="font-medium text-black mb-2">Is there API documentation?</h3>
                <p className="text-gray-700">
                  API documentation is in the code comments in <code className="bg-gray-200 px-2 py-1 rounded">cmd/server/</code>.
                  We're working on generating OpenAPI specs.
                </p>
              </div>

              <div>
                <h3 className="font-medium text-black mb-2">How do I deploy to production?</h3>
                <p className="text-gray-700">
                  See <code className="bg-gray-200 px-2 py-1 rounded">DEPLOYMENT.md</code> for complete deployment instructions
                  including Docker, Kubernetes, and VPS options.
                </p>
              </div>

              <div>
                <h3 className="font-medium text-black mb-2">Is there commercial support available?</h3>
                <p className="text-gray-700">
                  Yes, email <a href="mailto:contact@ocx.world" className="underline">contact@ocx.world</a> to discuss
                  enterprise support options.
                </p>
              </div>
            </div>
          </div>

          <div className="border-l-4 border-black pl-6">
            <h3 className="font-medium text-black mb-2">Self-Hosted Deployment</h3>
            <p className="text-gray-700">
              OCX Protocol is open source (MIT license). You can deploy and run it on your own infrastructure
              with full source code access. Community support is available via GitHub Issues.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Support;
