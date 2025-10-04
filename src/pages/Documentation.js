import React from 'react';
import { Book, Code, Github, FileText } from 'lucide-react';

const Documentation = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <h1 className="text-5xl font-light tracking-tight text-black mb-8">
          Documentation
        </h1>

        <div className="space-y-12">
          <p className="text-xl text-gray-700 leading-relaxed">
            Get started with OCX Protocol. All documentation is maintained in the
            GitHub repository with inline code examples.
          </p>

          <div className="grid md:grid-cols-2 gap-8">
            <a
              href="https://github.com/your-org/ocx-protocol#quick-start"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Code className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">Quick Start</h2>
              <p className="text-gray-600">Get running in 5 minutes with the demo script</p>
            </a>

            <a
              href="https://github.com/your-org/ocx-protocol/tree/main/cmd"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <FileText className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">API Reference</h2>
              <p className="text-gray-600">Inline documentation in cmd/server/</p>
            </a>

            <a
              href="https://github.com/your-org/ocx-protocol/tree/main/pkg"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Book className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">Package Docs</h2>
              <p className="text-gray-600">Go package documentation with examples</p>
            </a>

            <a
              href="https://github.com/your-org/ocx-protocol/tree/main/libocx-verify"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Github className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">Rust Verifier</h2>
              <p className="text-gray-600">Standalone verification library docs</p>
            </a>
          </div>

          <div className="bg-black text-green-400 p-6 rounded font-mono text-sm overflow-x-auto">
            <div className="text-gray-500"># Quick start</div>
            <div>git clone https://github.com/your-org/ocx-protocol</div>
            <div>cd ocx-protocol</div>
            <div>OCX_API_KEY=test OCX_PORT=9001 ./demo/DEMO.sh</div>
          </div>

          <div className="mt-12">
            <h2 className="text-2xl font-medium text-black mb-6">What you'll find</h2>
            <ul className="space-y-3 text-gray-700">
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>README.md</strong> - Main documentation with setup instructions</span>
              </li>
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>demo/DEMO.sh</strong> - Working demo showing execution and verification</span>
              </li>
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>scripts/smoke.sh</strong> - Comprehensive smoke test</span>
              </li>
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>pkg/*/</strong> - Package-level documentation with examples</span>
              </li>
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>k8s/</strong> - Kubernetes deployment configurations</span>
              </li>
              <li className="flex items-start">
                <span className="text-black mr-3">•</span>
                <span><strong>Dockerfile</strong> - Container deployment setup</span>
              </li>
            </ul>
          </div>

          <div className="bg-gray-50 p-8 rounded mt-12">
            <h3 className="font-medium text-black mb-4">Need help?</h3>
            <p className="text-gray-700 mb-4">
              If the documentation doesn't answer your question, open an issue on GitHub
              or email <a href="mailto:contact@ocx.world" className="underline">contact@ocx.world</a>.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Documentation;
