import React from 'react';
import { Mail, Github, FileText } from 'lucide-react';

const Contact = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <h1 className="text-5xl font-light tracking-tight text-black mb-8">
          Contact
        </h1>

        <div className="space-y-12">
          <p className="text-xl text-gray-700 leading-relaxed">
            Have questions about OCX Protocol? Want to discuss a deployment? Reach out.
          </p>

          <div className="grid md:grid-cols-2 gap-8">
            <a
              href="mailto:contact@ocx.world"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Mail className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">Email</h2>
              <p className="text-gray-600">contact@ocx.world</p>
            </a>

            <a
              href="https://github.com/KuroKernel/ocx-protocol"
              target="_blank"
              rel="noopener noreferrer"
              className="border-2 border-gray-200 p-8 rounded hover:border-black transition-colors group"
            >
              <Github className="w-10 h-10 text-black mb-4" />
              <h2 className="text-xl font-medium text-black mb-2 group-hover:underline">GitHub</h2>
              <p className="text-gray-600">Open issues and discussions</p>
            </a>
          </div>

          <div className="bg-gray-50 p-8 rounded mt-12">
            <h2 className="text-2xl font-medium text-black mb-6">For commercial inquiries</h2>
            <p className="text-gray-700 mb-6">
              If you're interested in managed hosting, enterprise support, or have other
              commercial questions, send us an email with details about your use case.
            </p>
            <p className="text-gray-700">
              We typically respond within 1-2 business days.
            </p>
          </div>

          <div className="border-l-4 border-black pl-6">
            <h3 className="font-medium text-black mb-2">Technical documentation</h3>
            <p className="text-gray-700">
              Check the <a href="https://github.com/KuroKernel/ocx-protocol" className="underline">README</a> and
              inline code documentation for technical details, API reference, and deployment guides.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Contact;
