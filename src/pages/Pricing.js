import React from 'react';
import { Mail } from 'lucide-react';

const Pricing = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <h1 className="text-5xl font-light tracking-tight text-black mb-8">
          Pricing
        </h1>

        <div className="space-y-8">
          <p className="text-xl text-gray-700 leading-relaxed">
            OCX Protocol is currently in pilot release. We're working with select
            partners to refine the platform before general availability.
          </p>

          <div className="border-2 border-black p-10 rounded my-12">
            <h2 className="text-2xl font-medium text-black mb-6">Pilot Program</h2>

            <div className="space-y-4 mb-8">
              <div className="flex justify-between items-start border-b border-gray-200 pb-4">
                <div>
                  <p className="font-medium text-black">Self-hosted deployment</p>
                  <p className="text-sm text-gray-600 mt-1">Run OCX on your own infrastructure</p>
                </div>
                <span className="text-lg font-medium">Free</span>
              </div>

              <div className="flex justify-between items-start border-b border-gray-200 pb-4">
                <div>
                  <p className="font-medium text-black">Managed service</p>
                  <p className="text-sm text-gray-600 mt-1">We handle hosting and operations</p>
                </div>
                <span className="text-lg font-medium">Contact us</span>
              </div>

              <div className="flex justify-between items-start">
                <div>
                  <p className="font-medium text-black">Enterprise support</p>
                  <p className="text-sm text-gray-600 mt-1">Dedicated engineering support</p>
                </div>
                <span className="text-lg font-medium">Contact us</span>
              </div>
            </div>

            <a
              href="mailto:contact@ocx.world"
              className="inline-flex items-center px-6 py-3 bg-black text-white rounded hover:bg-gray-800 transition-colors"
            >
              <Mail className="w-5 h-5 mr-2" />
              Get in touch
            </a>
          </div>

          <h2 className="text-2xl font-medium text-black mt-12 mb-6">Self-hosted deployment</h2>
          <p className="text-gray-700 leading-relaxed mb-4">
            The OCX Protocol server and verification tools are open source (MIT license).
            You can deploy and run them on your own infrastructure at no cost.
          </p>
          <ul className="space-y-2 text-gray-700 mb-6">
            <li className="flex items-start">
              <span className="mr-3">•</span>
              <span>No usage limits</span>
            </li>
            <li className="flex items-start">
              <span className="mr-3">•</span>
              <span>Full control over data and keys</span>
            </li>
            <li className="flex items-start">
              <span className="mr-3">•</span>
              <span>Air-gapped deployment supported</span>
            </li>
            <li className="flex items-start">
              <span className="mr-3">•</span>
              <span>Community support via GitHub</span>
            </li>
          </ul>

          <h2 className="text-2xl font-medium text-black mt-12 mb-6">Managed service</h2>
          <p className="text-gray-700 leading-relaxed mb-4">
            We're building a managed service for teams that want the benefits of OCX
            without running their own infrastructure. If you're interested in being
            part of the pilot program, reach out.
          </p>

          <div className="bg-gray-50 p-8 rounded mt-12">
            <h3 className="font-medium text-black mb-4">Questions about pricing?</h3>
            <p className="text-gray-700 mb-6">
              Email us at <a href="mailto:contact@ocx.world" className="text-black underline">contact@ocx.world</a> and
              we'll help you figure out the best deployment option.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Pricing;
