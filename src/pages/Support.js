import React, { useState } from 'react';
import { Mail, MessageCircle, Book, Search, ArrowRight, CheckCircle, Clock, Users, Zap } from 'lucide-react';

const Support = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');

  const supportCategories = [
    { id: 'all', name: 'All Topics', count: 24 },
    { id: 'getting-started', name: 'Getting Started', count: 8 },
    { id: 'api', name: 'API & Integration', count: 6 },
    { id: 'billing', name: 'Billing & Plans', count: 4 },
    { id: 'troubleshooting', name: 'Troubleshooting', count: 6 }
  ];

  const faqItems = [
    {
      id: 1,
      category: 'getting-started',
      question: 'How do I get started with OCX Protocol?',
      answer: 'Getting started is easy! First, sign up for a free account, then download our SDK or use our REST API. Check out our Quick Start guide for a 5-minute setup tutorial.',
      tags: ['getting-started', 'setup', 'tutorial']
    },
    {
      id: 2,
      category: 'api',
      question: 'What is the difference between execute and verify endpoints?',
      answer: 'The execute endpoint runs your computation and generates a cryptographic receipt. The verify endpoint validates that receipt offline without needing to re-run the computation.',
      tags: ['api', 'endpoints', 'receipts']
    },
    {
      id: 3,
      category: 'billing',
      question: 'How is pricing calculated?',
      answer: 'Pricing is based on the number of verifications you perform. The free tier includes 1M verifications per month. Additional verifications cost $10 per 1M.',
      tags: ['billing', 'pricing', 'verifications']
    },
    {
      id: 4,
      category: 'troubleshooting',
      question: 'Why is my receipt verification failing?',
      answer: 'Receipt verification can fail for several reasons: invalid receipt format, corrupted data, or signature mismatch. Check our troubleshooting guide for detailed steps.',
      tags: ['troubleshooting', 'verification', 'errors']
    },
    {
      id: 5,
      category: 'api',
      question: 'What programming languages are supported?',
      answer: 'We provide official SDKs for JavaScript/Node.js, Python, and Go. Our REST API works with any language that can make HTTP requests.',
      tags: ['api', 'sdk', 'languages']
    },
    {
      id: 6,
      category: 'getting-started',
      question: 'How do I generate my first receipt?',
      answer: 'Use the /api/v1/execute endpoint with your base64-encoded code and input. The response will include a receipt_blob that you can verify later.',
      tags: ['getting-started', 'receipts', 'execute']
    }
  ];

  const filteredFaqs = faqItems.filter(item => {
    const matchesCategory = selectedCategory === 'all' || item.category === selectedCategory;
    const matchesSearch = searchQuery === '' || 
      item.question.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.answer.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.tags.some(tag => tag.toLowerCase().includes(searchQuery.toLowerCase()));
    return matchesCategory && matchesSearch;
  });

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            Support Center
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Find answers, get help, and connect with our support team. 
            We're here to help you succeed with OCX Protocol.
          </p>
        </div>

        {/* Search */}
        <div className="max-w-2xl mx-auto mb-16">
          <div className="relative">
            <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Search for help..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-12 pr-4 py-4 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent text-lg"
            />
          </div>
        </div>

        {/* Quick Actions */}
        <div className="grid md:grid-cols-3 gap-8 mb-20">
          <a 
            href="#contact" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Mail className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Contact Support</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Get personalized help from our support team
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>Send message</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="https://discord.gg/ocx-protocol" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <MessageCircle className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Community Chat</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Join our Discord for real-time community support
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>Join Discord</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>

          <a 
            href="#documentation" 
            className="bg-gray-50 p-8 rounded-sm hover:bg-gray-100 transition-colors group"
          >
            <div className="flex items-center space-x-4 mb-4">
              <Book className="w-8 h-8 text-black" />
              <h3 className="text-xl font-medium text-black">Documentation</h3>
            </div>
            <p className="text-gray-600 mb-4">
              Browse our comprehensive documentation
            </p>
            <div className="flex items-center text-black group-hover:text-gray-600">
              <span>View docs</span>
              <ArrowRight className="w-4 h-4 ml-2" />
            </div>
          </a>
        </div>

        {/* Support Tiers */}
        <div className="mb-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12 text-center">
            Support Tiers
          </h2>
          
          <div className="grid md:grid-cols-3 gap-8">
            <div className="border border-gray-200 rounded-sm p-8">
              <div className="text-center mb-6">
                <div className="w-16 h-16 bg-gray-100 rounded-full mx-auto mb-4 flex items-center justify-center">
                  <Users className="w-8 h-8 text-gray-400" />
                </div>
                <h3 className="text-xl font-medium text-black mb-2">Community</h3>
                <div className="text-3xl font-light text-black mb-2">Free</div>
                <p className="text-gray-600">For development and testing</p>
              </div>
              <ul className="space-y-3 mb-8">
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Discord community support
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Documentation and guides
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  GitHub issues
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Email support (2 business days)
                </li>
              </ul>
            </div>

            <div className="border-2 border-black rounded-sm p-8 relative">
              <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                <span className="bg-black text-white px-4 py-2 rounded-sm text-sm font-medium">
                  RECOMMENDED
                </span>
              </div>
              <div className="text-center mb-6">
                <div className="w-16 h-16 bg-black rounded-full mx-auto mb-4 flex items-center justify-center">
                  <Zap className="w-8 h-8 text-white" />
                </div>
                <h3 className="text-xl font-medium text-black mb-2">Professional</h3>
                <div className="text-3xl font-light text-black mb-2">$299<span className="text-lg text-gray-600">/mo</span></div>
                <p className="text-gray-600">For growing teams</p>
              </div>
              <ul className="space-y-3 mb-8">
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Priority email support (1 business day)
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Discord community access
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Phone support
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Custom integrations
                </li>
              </ul>
            </div>

            <div className="border border-gray-200 rounded-sm p-8">
              <div className="text-center mb-6">
                <div className="w-16 h-16 bg-gray-100 rounded-full mx-auto mb-4 flex items-center justify-center">
                  <Clock className="w-8 h-8 text-gray-400" />
                </div>
                <h3 className="text-xl font-medium text-black mb-2">Enterprise</h3>
                <div className="text-3xl font-light text-black mb-2">Custom</div>
                <p className="text-gray-600">For large organizations</p>
              </div>
              <ul className="space-y-3 mb-8">
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Dedicated support team
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  24/7 phone and email support
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  On-premises deployment support
                </li>
                <li className="flex items-center text-gray-600">
                  <CheckCircle className="w-4 h-4 text-green-600 mr-3 flex-shrink-0" />
                  Custom SLA guarantees
                </li>
              </ul>
            </div>
          </div>
        </div>

        {/* FAQ Categories */}
        <div className="mb-12">
          <h2 className="text-3xl font-light tracking-tight text-black mb-8">Frequently Asked Questions</h2>
          
          <div className="flex flex-wrap gap-4 mb-8">
            {supportCategories.map((category) => (
              <button
                key={category.id}
                onClick={() => setSelectedCategory(category.id)}
                className={`px-4 py-2 rounded-sm text-sm font-medium transition-colors ${
                  selectedCategory === category.id
                    ? 'bg-black text-white'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }`}
              >
                {category.name} ({category.count})
              </button>
            ))}
          </div>
        </div>

        {/* FAQ Items */}
        <div className="space-y-6 mb-20">
          {filteredFaqs.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 rounded-sm">
              <Search className="w-16 h-16 text-gray-400 mx-auto mb-4" />
              <h3 className="text-xl font-medium text-black mb-2">No results found</h3>
              <p className="text-gray-600">
                Try adjusting your search terms or browse different categories.
              </p>
            </div>
          ) : (
            filteredFaqs.map((item) => (
              <div key={item.id} className="border border-gray-200 rounded-sm p-6">
                <h3 className="text-lg font-medium text-black mb-3">{item.question}</h3>
                <p className="text-gray-600 mb-4">{item.answer}</p>
                <div className="flex flex-wrap gap-2">
                  {item.tags.map((tag, index) => (
                    <span
                      key={index}
                      className="px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            ))
          )}
        </div>

        {/* Contact Support */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Still need help?
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Our support team is here to help. Get in touch and we'll respond quickly.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <a 
              href="#contact" 
              className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center"
            >
              Contact Support
              <ArrowRight className="w-5 h-5 ml-3" />
            </a>
            <a 
              href="https://discord.gg/ocx-protocol" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              Join Discord
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Support;
