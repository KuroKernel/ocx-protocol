import React from 'react';
import { Mail, Check, Code, Server, Building } from 'lucide-react';

const Pricing = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-16">
          <h1 className="text-5xl font-light tracking-tight text-black mb-8">
            Deployment Options
          </h1>
          <p className="text-xl text-gray-700 leading-relaxed max-w-3xl mx-auto">
            Choose the deployment model that fits your needs. From open-source self-hosting
            to fully managed enterprise solutions.
          </p>
        </div>

        <div className="grid lg:grid-cols-3 gap-8 mb-16">
          <div className="border border-gray-200 rounded-sm p-8">
            <div className="flex items-center space-x-3 mb-6">
              <Code className="w-8 h-8 text-black" />
              <h2 className="text-2xl font-medium text-black">Self-Hosted</h2>
            </div>
            <div className="text-3xl font-light text-black mb-6">Open Source</div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">MIT licensed, free forever</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">No usage limits or restrictions</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Full control over data and keys</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Air-gapped deployment supported</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Community support via GitHub</span>
              </li>
            </ul>
            <a
              href="https://github.com/KuroKernel/ocx-protocol"
              target="_blank"
              rel="noopener noreferrer"
              className="block w-full text-center border border-black text-black py-3 rounded hover:bg-black hover:text-white transition-colors"
            >
              View on GitHub
            </a>
          </div>

          <div className="border-2 border-black rounded-sm p-8 relative bg-white">
            <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
              <span className="bg-black text-white px-6 py-2 rounded-sm text-sm font-medium">MANAGED</span>
            </div>
            <div className="flex items-center space-x-3 mb-6">
              <Server className="w-8 h-8 text-black" />
              <h2 className="text-2xl font-medium text-black">Hosted Service</h2>
            </div>
            <div className="text-3xl font-light text-black mb-6">Contact Us</div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Fully managed infrastructure</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Automatic scaling and updates</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">99.9% uptime SLA</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">24/7 monitoring and support</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Priority feature requests</span>
              </li>
            </ul>
            <a
              href="mailto:contact@ocx.world"
              className="block w-full text-center bg-black text-white py-3 rounded hover:bg-gray-800 transition-colors"
            >
              Request Pricing
            </a>
          </div>

          <div className="border border-gray-200 rounded-sm p-8">
            <div className="flex items-center space-x-3 mb-6">
              <Building className="w-8 h-8 text-black" />
              <h2 className="text-2xl font-medium text-black">Enterprise</h2>
            </div>
            <div className="text-3xl font-light text-black mb-6">Custom</div>
            <ul className="space-y-3 mb-8">
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">On-premises deployment</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Custom SLA agreements</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Dedicated engineering support</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Integration assistance</span>
              </li>
              <li className="flex items-start">
                <Check className="w-5 h-5 text-black mr-3 mt-0.5 flex-shrink-0" />
                <span className="text-gray-700">Training and onboarding</span>
              </li>
            </ul>
            <a
              href="mailto:contact@ocx.world"
              className="block w-full text-center border border-black text-black py-3 rounded hover:bg-black hover:text-white transition-colors"
            >
              Contact Sales
            </a>
          </div>
        </div>

        <div className="bg-gray-50 p-12 rounded-sm text-center">
          <h3 className="text-2xl font-medium text-black mb-4">Questions about deployment?</h3>
          <p className="text-gray-700 mb-8 max-w-2xl mx-auto">
            We're happy to help you choose the right option for your use case.
            Email us at <a href="mailto:contact@ocx.world" className="text-black underline font-medium">contact@ocx.world</a> or
            check out our documentation to get started.
          </p>
          <div className="flex items-center justify-center space-x-4">
            <a
              href="mailto:contact@ocx.world"
              className="inline-flex items-center px-6 py-3 bg-black text-white rounded hover:bg-gray-800 transition-colors"
            >
              <Mail className="w-5 h-5 mr-2" />
              Get in Touch
            </a>
            <a
              href="https://github.com/KuroKernel/ocx-protocol"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center px-6 py-3 border border-black text-black rounded hover:bg-black hover:text-white transition-colors"
            >
              <Code className="w-5 h-5 mr-2" />
              View Documentation
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Pricing;
