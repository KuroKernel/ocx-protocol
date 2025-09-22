import React, { useState } from 'react';
import { Mail, Phone, MapPin, Send, ArrowRight, CheckCircle } from 'lucide-react';

const Contact = () => {
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    company: '',
    subject: '',
    message: '',
    interest: 'general'
  });
  const [isSubmitted, setIsSubmitted] = useState(false);

  const handleInputChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value
    });
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    // Simulate form submission
    setIsSubmitted(true);
    setTimeout(() => setIsSubmitted(false), 3000);
  };

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            Contact Us
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Get in touch with our team. We're here to help you integrate 
            OCX Protocol into your applications.
          </p>
        </div>

        <div className="grid lg:grid-cols-2 gap-16">
          {/* Contact Form */}
          <div>
            <h2 className="text-3xl font-light tracking-tight text-black mb-8">
              Send us a message
            </h2>
            
            {isSubmitted ? (
              <div className="bg-green-50 border border-green-200 rounded-sm p-8 text-center">
                <CheckCircle className="w-16 h-16 text-green-600 mx-auto mb-4" />
                <h3 className="text-xl font-medium text-black mb-2">Message sent!</h3>
                <p className="text-gray-600">
                  We'll get back to you within 24 hours.
                </p>
              </div>
            ) : (
              <form onSubmit={handleSubmit} className="space-y-6">
                <div className="grid md:grid-cols-2 gap-6">
                  <div>
                    <label htmlFor="name" className="block text-sm font-medium text-black mb-2">
                      Name *
                    </label>
                    <input
                      type="text"
                      id="name"
                      name="name"
                      value={formData.name}
                      onChange={handleInputChange}
                      required
                      className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                      placeholder="Your name"
                    />
                  </div>
                  
                  <div>
                    <label htmlFor="email" className="block text-sm font-medium text-black mb-2">
                      Email *
                    </label>
                    <input
                      type="email"
                      id="email"
                      name="email"
                      value={formData.email}
                      onChange={handleInputChange}
                      required
                      className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                      placeholder="your@email.com"
                    />
                  </div>
                </div>
                
                <div>
                  <label htmlFor="company" className="block text-sm font-medium text-black mb-2">
                    Company
                  </label>
                  <input
                    type="text"
                    id="company"
                    name="company"
                    value={formData.company}
                    onChange={handleInputChange}
                    className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                    placeholder="Your company"
                  />
                </div>
                
                <div>
                  <label htmlFor="interest" className="block text-sm font-medium text-black mb-2">
                    I'm interested in
                  </label>
                  <select
                    id="interest"
                    name="interest"
                    value={formData.interest}
                    onChange={handleInputChange}
                    className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                  >
                    <option value="general">General inquiry</option>
                    <option value="pilot">Pilot program</option>
                    <option value="enterprise">Enterprise sales</option>
                    <option value="partnership">Partnership</option>
                    <option value="support">Technical support</option>
                    <option value="media">Media inquiry</option>
                  </select>
                </div>
                
                <div>
                  <label htmlFor="subject" className="block text-sm font-medium text-black mb-2">
                    Subject *
                  </label>
                  <input
                    type="text"
                    id="subject"
                    name="subject"
                    value={formData.subject}
                    onChange={handleInputChange}
                    required
                    className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                    placeholder="What's this about?"
                  />
                </div>
                
                <div>
                  <label htmlFor="message" className="block text-sm font-medium text-black mb-2">
                    Message *
                  </label>
                  <textarea
                    id="message"
                    name="message"
                    value={formData.message}
                    onChange={handleInputChange}
                    required
                    rows={6}
                    className="w-full px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
                    placeholder="Tell us more about your inquiry..."
                  />
                </div>
                
                <button
                  type="submit"
                  className="w-full bg-black text-white py-4 rounded-sm hover:bg-gray-900 transition-colors font-medium flex items-center justify-center"
                >
                  <Send className="w-5 h-5 mr-2" />
                  Send Message
                </button>
              </form>
            )}
          </div>

          {/* Contact Information */}
          <div>
            <h2 className="text-3xl font-light tracking-tight text-black mb-8">
              Get in touch
            </h2>
            
            <div className="space-y-8 mb-12">
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <Mail className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-medium text-black mb-2">Email</h3>
                  <p className="text-gray-600 mb-2">General inquiries</p>
                  <a href="mailto:hello@ocx-protocol.com" className="text-black hover:text-gray-600">
                    hello@ocx-protocol.com
                  </a>
                </div>
              </div>
              
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <Phone className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-medium text-black mb-2">Phone</h3>
                  <p className="text-gray-600 mb-2">Enterprise sales</p>
                  <a href="tel:+1-555-123-4567" className="text-black hover:text-gray-600">
                    +1 (555) 123-4567
                  </a>
                </div>
              </div>
              
              <div className="flex items-start space-x-4">
                <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                  <MapPin className="w-4 h-4 text-white" />
                </div>
                <div>
                  <h3 className="text-lg font-medium text-black mb-2">Office</h3>
                  <p className="text-gray-600 mb-2">San Francisco, CA</p>
                  <p className="text-gray-600">
                    1234 Protocol Street<br />
                    San Francisco, CA 94105
                  </p>
                </div>
              </div>
            </div>

            {/* Response Times */}
            <div className="bg-gray-50 p-6 rounded-sm">
              <h3 className="text-lg font-medium text-black mb-4">Response Times</h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">General inquiries</span>
                  <span className="font-medium text-black">24 hours</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Pilot program</span>
                  <span className="font-medium text-black">4 hours</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Enterprise sales</span>
                  <span className="font-medium text-black">2 hours</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Technical support</span>
                  <span className="font-medium text-black">1 hour</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Additional Contact Options */}
        <div className="mt-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12 text-center">
            Other ways to connect
          </h2>
          
          <div className="grid md:grid-cols-3 gap-8">
            <a 
              href="https://discord.gg/ocx-protocol" 
              className="border border-gray-200 rounded-sm p-8 hover:bg-gray-50 transition-colors group text-center"
            >
              <div className="w-12 h-12 bg-black rounded-sm mx-auto mb-4 flex items-center justify-center">
                <span className="text-white font-bold">D</span>
              </div>
              <h3 className="text-xl font-medium text-black mb-2 group-hover:text-gray-600">
                Discord Community
              </h3>
              <p className="text-gray-600 mb-4">
                Join our Discord server for real-time chat and community support
              </p>
              <div className="flex items-center justify-center text-black group-hover:text-gray-600">
                <span>Join Discord</span>
                <ArrowRight className="w-4 h-4 ml-2" />
              </div>
            </a>

            <a 
              href="https://github.com/ocx-protocol/ocx" 
              className="border border-gray-200 rounded-sm p-8 hover:bg-gray-50 transition-colors group text-center"
            >
              <div className="w-12 h-12 bg-black rounded-sm mx-auto mb-4 flex items-center justify-center">
                <span className="text-white font-bold">G</span>
              </div>
              <h3 className="text-xl font-medium text-black mb-2 group-hover:text-gray-600">
                GitHub
              </h3>
              <p className="text-gray-600 mb-4">
                Contribute to our open source project and report issues
              </p>
              <div className="flex items-center justify-center text-black group-hover:text-gray-600">
                <span>View on GitHub</span>
                <ArrowRight className="w-4 h-4 ml-2" />
              </div>
            </a>

            <a 
              href="https://status.ocx-protocol.com" 
              className="border border-gray-200 rounded-sm p-8 hover:bg-gray-50 transition-colors group text-center"
            >
              <div className="w-12 h-12 bg-black rounded-sm mx-auto mb-4 flex items-center justify-center">
                <span className="text-white font-bold">S</span>
              </div>
              <h3 className="text-xl font-medium text-black mb-2 group-hover:text-gray-600">
                Status Page
              </h3>
              <p className="text-gray-600 mb-4">
                Check service status and subscribe to incident updates
              </p>
              <div className="flex items-center justify-center text-black group-hover:text-gray-600">
                <span>Check Status</span>
                <ArrowRight className="w-4 h-4 ml-2" />
              </div>
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Contact;
