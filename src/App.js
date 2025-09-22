import React, { useState, useEffect } from 'react';
import { ArrowRight, Check, Hash } from 'lucide-react';
import Specification from './pages/Specification';
import Pricing from './pages/Pricing';
import Documentation from './pages/Documentation';
import About from './pages/About';
import Contact from './pages/Contact';
import APIReference from './pages/APIReference';
import Status from './pages/Status';
import Support from './pages/Support';
import TestPage from './pages/TestPage';

const API_BASE = process.env.NODE_ENV === 'production' ? '' : 'http://localhost:3001';

const OCXLanding = () => {
  const [currentPage, setCurrentPage] = useState('home');
  const [demoResult, setDemoResult] = useState(null);
  const [verifyResult, setVerifyResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [receipts, setReceipts] = useState([]);
  const [executionForm, setExecutionForm] = useState({
    artifact: 'aGVsbG93b3JsZA==',
    input: 'dGVzdGRhdGE=',
    max_cycles: 10000
  });
  const [verifyForm, setVerifyForm] = useState({
    receipt_blob: ''
  });

  // Fetch receipts on component mount
  useEffect(() => {
    fetchReceipts();
  }, []);

  const fetchReceipts = async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/receipts`);
      const data = await response.json();
      setReceipts(data.receipts || []);
    } catch (error) {
      console.error('Failed to fetch receipts:', error);
    }
  };

  const handleExecute = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/api/v1/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(executionForm)
      });
      const result = await response.json();
      setDemoResult(result);
    } catch (error) {
      console.error('Execution failed:', error);
      setDemoResult({ error: 'Failed to execute. Please check if the API server is running.' });
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE}/api/v1/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(verifyForm)
      });
      const result = await response.json();
      setVerifyResult(result);
    } catch (error) {
      console.error('Verification failed:', error);
      setVerifyResult({ valid: false, reason: 'API connection failed' });
    } finally {
      setLoading(false);
    }
  };

  const navigateToPage = (page) => {
    console.log('Navigating to page:', page);
    setCurrentPage(page);
    window.scrollTo(0, 0);
  };

  const renderPage = () => {
    console.log('Current page:', currentPage);
    if (currentPage === 'specification') {
      return <Specification />;
    } else if (currentPage === 'pricing') {
      return <Pricing />;
    } else if (currentPage === 'documentation') {
      return <Documentation />;
    } else if (currentPage === 'about') {
      return <About />;
    } else if (currentPage === 'contact') {
      return <Contact />;
    } else if (currentPage === 'api-reference') {
      return <APIReference />;
    } else if (currentPage === 'status') {
      return <Status />;
    } else if (currentPage === 'support') {
      return <Support />;
    } else if (currentPage === 'test') {
      return <TestPage />;
    } else {
      return renderHomeContent();
    }
  };

  const renderHomeContent = () => {
    return (
      <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="fixed top-0 w-full bg-white/90 backdrop-blur-sm border-b border-gray-100 z-50">
        <div className="max-w-6xl mx-auto px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3">
              <img 
                src="/assets/logos/ocx-symbol-only.svg" 
                alt="OCX Protocol" 
                className="w-8 h-8"
              />
              <span className="text-lg font-medium tracking-tight">OCX Protocol</span>
            </div>
            <nav className="hidden md:flex items-center space-x-12">
              <button 
                onClick={() => navigateToPage('specification')} 
                className="text-gray-600 hover:text-black transition-colors"
              >
                Specification
              </button>
              <button 
                onClick={() => navigateToPage('pricing')} 
                className="text-gray-600 hover:text-black transition-colors"
              >
                Pricing
              </button>
              <button 
                onClick={() => navigateToPage('documentation')} 
                className="text-gray-600 hover:text-black transition-colors"
              >
                Documentation
              </button>
              <button 
                onClick={() => navigateToPage('contact')}
                className="bg-black text-white px-6 py-2 rounded-sm hover:bg-gray-900 transition-colors"
              >
                Start Building
              </button>
            </nav>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="pt-40 pb-32">
        <div className="max-w-6xl mx-auto px-8">
          <div className="max-w-4xl">
            <h1 className="text-7xl font-light tracking-tight text-black mb-8 leading-[0.9]">
              Computational<br />
              integrity<br />
              through<br />
              mathematical proof
            </h1>
            <p className="text-xl text-gray-600 mb-16 leading-relaxed max-w-2xl">
              Every execution generates cryptographic receipts. Offline verification in milliseconds. 
              Transform disputes into mathematical certainty.
            </p>
            
            <div className="flex items-center space-x-8">
              <button 
                onClick={() => navigateToPage('contact')}
                className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-all duration-200 flex items-center text-lg"
              >
                Get Started
                <ArrowRight className="w-5 h-5 ml-3" />
              </button>
              <button 
                onClick={() => navigateToPage('specification')} 
                className="text-gray-600 hover:text-black transition-colors text-lg border-b border-gray-300 pb-1"
              >
                Read Specification
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Value Proposition */}
      <section className="py-32 bg-gray-50 relative overflow-hidden">
        {/* Subtle geometric elements */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-32 right-32 w-16 h-16 border border-gray-300 rotate-45 opacity-10"></div>
          <div className="absolute bottom-20 left-20 w-12 h-12 bg-gray-200 rounded-full opacity-20"></div>
          <div className="absolute top-1/2 right-1/4 w-8 h-8 border-2 border-gray-400 transform -rotate-12 opacity-15"></div>
        </div>
        
        <div className="max-w-6xl mx-auto px-8 relative">
          <div className="grid lg:grid-cols-2 gap-20 items-center">
            <div>
              <h2 className="text-5xl font-light tracking-tight text-black mb-12 leading-tight">
                Replace institutional trust with mathematical certainty
              </h2>
              <div className="space-y-8">
                <div className="flex items-start space-x-4">
                  <div className="w-6 h-6 border-2 border-black rounded-sm flex-shrink-0 mt-1"></div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-4">Deterministic Execution</h3>
                    <p className="text-gray-600 leading-relaxed text-lg">
                      Isolated runtime environment ensures identical results across all architectures. 
                      No syscalls, no floating point, no non-determinism.
                    </p>
                  </div>
                </div>
                
                <div className="flex items-start space-x-4">
                  <div className="w-6 h-6 bg-black rounded-sm flex-shrink-0 mt-1"></div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-4">Cryptographic Receipts</h3>
                    <p className="text-gray-600 leading-relaxed text-lg">
                      SHA-256 hashing with Ed25519 signatures. CBOR encoding for 
                      universal compatibility. Immutable proof of execution.
                    </p>
                  </div>
                </div>
                
                <div className="flex items-start space-x-4">
                  <div className="w-6 h-6 border-2 border-gray-400 rounded-full flex-shrink-0 mt-1"></div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-4">Offline Verification</h3>
                    <p className="text-gray-600 leading-relaxed text-lg">
                      Air-gapped validation without network dependency. 
                      Mathematical verification in milliseconds, anywhere.
                    </p>
                  </div>
                </div>
              </div>
            </div>
            
            {/* OCX Graphics */}
            <div className="w-full h-96 md:h-80 lg:h-96 rounded-lg overflow-hidden shadow-lg flex items-center justify-center">
              <img 
                src="/assets/ocxgraphics.gif" 
                alt="OCX Protocol Graphics" 
                className="w-full h-full object-cover rounded-lg"
              />
            </div>
          </div>
        </div>
      </section>

      {/* Protocol Demonstration */}
      <section className="py-32 relative overflow-hidden">
        {/* Geometric Background Elements */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-20 right-20 w-32 h-32 border border-gray-200 rotate-45 opacity-20"></div>
          <div className="absolute bottom-40 left-16 w-24 h-24 bg-gray-100 rounded-full opacity-30"></div>
          <div className="absolute top-1/2 left-1/4 w-16 h-16 border-2 border-gray-300 transform rotate-12 opacity-25"></div>
        </div>
        
        <div className="max-w-6xl mx-auto px-8 relative">
          <div className="text-center mb-20">
            <h2 className="text-5xl font-light tracking-tight text-black mb-8">
              Cryptographic execution
            </h2>
            <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
              Every computation generates an immutable receipt. 
              Mathematical proof replaces institutional trust.
            </p>
          </div>
          
          <div className="grid lg:grid-cols-3 gap-16 items-center">
            {/* Input */}
            <div className="text-center">
              <div className="w-20 h-20 mx-auto mb-8 relative">
                <div className="w-full h-full border-2 border-gray-300 rounded-lg flex items-center justify-center">
                  <div className="w-8 h-8 bg-gray-200 rounded"></div>
                </div>
                <div className="absolute -top-2 -right-2 w-6 h-6 bg-black rounded-full flex items-center justify-center">
                  <div className="w-2 h-2 bg-white rounded-full"></div>
                </div>
              </div>
              <h3 className="text-xl font-medium text-black mb-4">Input</h3>
              <p className="text-gray-600 leading-relaxed">
                Deterministic execution environment processes your computation with mathematical precision.
              </p>
            </div>

            {/* Process */}
            <div className="text-center">
              <div className="w-20 h-20 mx-auto mb-8 relative">
                <div className="w-full h-full bg-black rounded-lg flex items-center justify-center">
                  <div className="w-8 h-8 border-2 border-white rounded"></div>
                </div>
                <div className="absolute -bottom-2 -left-2 w-6 h-6 bg-gray-200 rounded-full flex items-center justify-center">
                  <div className="w-2 h-2 bg-gray-600 rounded-full"></div>
                </div>
              </div>
              <h3 className="text-xl font-medium text-black mb-4">Execution</h3>
              <p className="text-gray-600 leading-relaxed">
                Isolated runtime generates cryptographic proof of computation integrity and resource usage.
              </p>
            </div>

            {/* Output */}
            <div className="text-center">
              <div className="w-20 h-20 mx-auto mb-8 relative">
                <div className="w-full h-full border-2 border-green-200 bg-green-50 rounded-lg flex items-center justify-center">
                  <div className="w-8 h-8 bg-green-600 rounded"></div>
                </div>
                <div className="absolute -top-2 -left-2 w-6 h-6 bg-green-600 rounded-full flex items-center justify-center">
                  <div className="w-2 h-2 bg-white rounded-full"></div>
                </div>
              </div>
              <h3 className="text-xl font-medium text-black mb-4">Receipt</h3>
              <p className="text-gray-600 leading-relaxed">
                Immutable cryptographic receipt enables offline verification and dispute resolution.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Technical Standards */}
      <section className="py-32 bg-gray-50 relative overflow-hidden">
        {/* Geometric background elements */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-16 left-16 w-20 h-20 border border-gray-300 rotate-45 opacity-15"></div>
          <div className="absolute bottom-32 right-24 w-14 h-14 bg-gray-200 rounded-full opacity-25"></div>
          <div className="absolute top-1/3 right-16 w-10 h-10 border-2 border-gray-400 transform rotate-12 opacity-20"></div>
        </div>
        
        <div className="max-w-6xl mx-auto px-8 relative">
          <h2 className="text-5xl font-light tracking-tight text-black mb-20 text-center max-w-4xl mx-auto">
            Built on open standards
          </h2>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-12">
            <div className="text-center relative">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 relative">
                <div className="absolute -top-1 -right-1 w-4 h-4 bg-gray-300 rounded-full"></div>
              </div>
              <h3 className="font-medium text-black mb-3">RFC 7049</h3>
              <p className="text-gray-600">CBOR encoding with canonical serialization</p>
            </div>
            
            <div className="text-center relative">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 relative">
                <div className="absolute -bottom-1 -left-1 w-3 h-3 bg-gray-400 rounded"></div>
              </div>
              <h3 className="font-medium text-black mb-3">RFC 8032</h3>
              <p className="text-gray-600">Ed25519 cryptographic signatures</p>
            </div>
            
            <div className="text-center relative">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 relative">
                <div className="absolute -top-1 -left-1 w-4 h-4 border border-gray-300 rounded"></div>
              </div>
              <h3 className="font-medium text-black mb-3">Cross-Platform</h3>
              <p className="text-gray-600">Identical execution on x86_64 and ARM64</p>
            </div>
            
            <div className="text-center relative">
              <div className="w-16 h-16 bg-black rounded-sm mx-auto mb-6 relative">
                <div className="absolute -bottom-1 -right-1 w-3 h-3 bg-gray-200 rounded-full"></div>
              </div>
              <h3 className="font-medium text-black mb-3">Cycle Metering</h3>
              <p className="text-gray-600">Precise computational resource measurement</p>
            </div>
          </div>
        </div>
      </section>

      {/* Use Cases */}
      <section className="py-32 relative overflow-hidden">
        {/* Geometric background elements */}
        <div className="absolute inset-0 pointer-events-none">
          <div className="absolute top-24 right-32 w-18 h-18 border border-gray-200 rotate-45 opacity-10"></div>
          <div className="absolute bottom-40 left-32 w-12 h-12 bg-gray-100 rounded-full opacity-20"></div>
          <div className="absolute top-1/2 left-1/3 w-8 h-8 border-2 border-gray-300 transform -rotate-12 opacity-15"></div>
        </div>
        
        <div className="max-w-6xl mx-auto px-8 relative">
          <h2 className="text-5xl font-light tracking-tight text-black mb-20 text-center">
            Enterprise applications
          </h2>
          
          <div className="grid lg:grid-cols-3 gap-16">
            <div className="relative">
              <div className="absolute -top-4 -left-4 w-6 h-6 border border-gray-300 rounded-sm opacity-30"></div>
              <h3 className="text-2xl font-medium text-black mb-6">Financial Services</h3>
              <p className="text-gray-600 leading-relaxed text-lg mb-6">
                Transform model validation and regulatory compliance. Replace months-long audits 
                with cryptographic proof of correct execution.
              </p>
              <ul className="space-y-3">
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Risk model validation
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  SOX compliance evidence
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Algorithmic trading verification
                </li>
              </ul>
            </div>
            
            <div className="relative">
              <div className="absolute -top-4 -right-4 w-6 h-6 bg-gray-200 rounded-full opacity-30"></div>
              <h3 className="text-2xl font-medium text-black mb-6">Machine Learning</h3>
              <p className="text-gray-600 leading-relaxed text-lg mb-6">
                Eliminate "works on my machine" problems. Provide reproducible, 
                verifiable model training and inference at scale.
              </p>
              <ul className="space-y-3">
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Reproducible experiments
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Model evaluation audits
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Training provenance
                </li>
              </ul>
            </div>
            
            <div className="relative">
              <div className="absolute -bottom-4 -left-4 w-6 h-6 border-2 border-gray-400 rounded opacity-30"></div>
              <h3 className="text-2xl font-medium text-black mb-6">Media & Content</h3>
              <p className="text-gray-600 leading-relaxed text-lg mb-6">
                Establish verifiable chain of custody for digital assets. 
                Resolve disputes with mathematical proof, not lengthy litigation.
              </p>
              <ul className="space-y-3">
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Content transformation proof
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Processing pipeline integrity
                </li>
                <li className="flex items-center text-gray-600">
                  <div className="w-2 h-2 bg-black rounded-full mr-3 flex-shrink-0"></div>
                  Watermark verification
                </li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" className="py-32 bg-gray-50">
        <div className="max-w-6xl mx-auto px-8">
          <h2 className="text-5xl font-light tracking-tight text-black mb-20 text-center">
            Transparent pricing
          </h2>
          
          <div className="grid lg:grid-cols-3 gap-8 max-w-5xl mx-auto">
            <div className="border border-gray-200 rounded-sm p-10">
              <h3 className="text-xl font-medium text-black mb-3">Development</h3>
              <div className="text-4xl font-light text-black mb-8">Free</div>
              <ul className="space-y-4 mb-10">
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  1M verifications/month
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  100k receipts stored
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  Community support
                </li>
              </ul>
              <button 
                onClick={() => navigateToPage('contact')}
                className="w-full border border-black text-black py-4 rounded-sm hover:bg-black hover:text-white transition-colors"
              >
                Start Building
              </button>
            </div>
            
            <div className="border-2 border-black rounded-sm p-10 relative bg-white">
              <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                <span className="bg-black text-white px-6 py-2 rounded-sm text-sm font-medium">RECOMMENDED</span>
              </div>
              <h3 className="text-xl font-medium text-black mb-3">Professional</h3>
              <div className="text-4xl font-light text-black mb-8">$299<span className="text-xl text-gray-600">/mo</span></div>
              <ul className="space-y-4 mb-10">
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  20M verifications/month
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  2M receipts stored
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  99.9% SLA
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  Priority support
                </li>
              </ul>
              <button 
                onClick={() => navigateToPage('contact')}
                className="w-full bg-black text-white py-4 rounded-sm hover:bg-gray-900 transition-colors"
              >
                Start Trial
              </button>
            </div>
            
            <div className="border border-gray-200 rounded-sm p-10">
              <h3 className="text-xl font-medium text-black mb-3">Enterprise</h3>
              <div className="text-4xl font-light text-black mb-8">Custom</div>
              <ul className="space-y-4 mb-10">
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  Unlimited verifications
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  On-premises deployment
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  SSO integration
                </li>
                <li className="flex items-center text-gray-600">
                  <Check className="w-4 h-4 text-black mr-4" />
                  Dedicated support
                </li>
              </ul>
              <button 
                onClick={() => navigateToPage('contact')}
                className="w-full border border-black text-black py-4 rounded-sm hover:bg-black hover:text-white transition-colors"
              >
                Contact Sales
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Final CTA */}
      <section className="py-32 bg-black text-white">
        <div className="max-w-6xl mx-auto px-8 text-center">
          <h2 className="text-5xl font-light tracking-tight mb-8 max-w-4xl mx-auto leading-tight">
            Deploy computational integrity across your infrastructure
          </h2>
          <p className="text-xl text-gray-400 mb-16 max-w-2xl mx-auto leading-relaxed">
            Start with a single endpoint. Scale to enterprise-wide verification. 
            Mathematical proof for every critical computation.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <button 
              onClick={() => navigateToPage('documentation')} 
              className="text-gray-400 hover:text-white transition-colors border-b border-gray-600 pb-1 text-lg"
            >
              View Documentation
            </button>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="py-20 bg-white border-t border-gray-100">
        <div className="max-w-6xl mx-auto px-8">
          <div className="grid md:grid-cols-4 gap-16">
            <div>
              <div className="flex items-center space-x-3 mb-8">
                <img 
                  src="/assets/logos/ocx-symbol-only.svg" 
                  alt="OCX Protocol" 
                  className="w-6 h-6"
                />
                <span className="text-lg font-medium tracking-tight">OCX Protocol</span>
              </div>
              <p className="text-gray-600 leading-relaxed">
                Mathematical proof for computational integrity
              </p>
            </div>
            
            <div>
              <h4 className="font-medium text-black mb-6">Product</h4>
              <ul className="space-y-4 text-gray-600">
                <li><button onClick={() => navigateToPage('specification')} className="hover:text-black transition-colors">Specification</button></li>
                <li><button onClick={() => navigateToPage('documentation')} className="hover:text-black transition-colors">Documentation</button></li>
                <li><button onClick={() => navigateToPage('api-reference')} className="hover:text-black transition-colors">API Reference</button></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-medium text-black mb-6">Company</h4>
              <ul className="space-y-4 text-gray-600">
                <li><button onClick={() => navigateToPage('about')} className="hover:text-black transition-colors">About</button></li>
                <li><button onClick={() => navigateToPage('specification')} className="hover:text-black transition-colors">Security</button></li>
                <li><button onClick={() => navigateToPage('contact')} className="hover:text-black transition-colors">Contact</button></li>
              </ul>
            </div>
            
            <div>
              <h4 className="font-medium text-black mb-6">Resources</h4>
              <ul className="space-y-4 text-gray-600">
                <li><a href="https://github.com/ocx-protocol/ocx" className="hover:text-black transition-colors">GitHub</a></li>
                <li><button onClick={() => navigateToPage('status')} className="hover:text-black transition-colors">Status</button></li>
                <li><button onClick={() => navigateToPage('support')} className="hover:text-black transition-colors">Support</button></li>
              </ul>
            </div>
          </div>
          
          <div className="border-t border-gray-100 mt-16 pt-8">
            <p className="text-center text-gray-500">
              © 2025 OCX Protocol. Mathematical proof for computational integrity.
            </p>
          </div>
        </div>
      </footer>
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-white">
      {/* Global Header */}
      <header className="fixed top-0 w-full bg-white/90 backdrop-blur-sm border-b border-gray-100 z-50">
        <div className="max-w-6xl mx-auto px-8 py-6">
          <div className="flex items-center justify-between">
            <button 
              onClick={() => navigateToPage('home')}
              className="flex items-center space-x-3 hover:opacity-80 transition-opacity"
            >
              <img 
                src="/assets/logos/ocx-symbol-only.svg" 
                alt="OCX Protocol" 
                className="w-8 h-8"
              />
              <span className="text-lg font-medium tracking-tight">OCX Protocol</span>
            </button>
            <nav className="hidden md:flex items-center space-x-12">
              <button 
                onClick={() => navigateToPage('specification')} 
                className={`transition-colors ${currentPage === 'specification' ? 'text-black' : 'text-gray-600 hover:text-black'}`}
              >
                Specification
              </button>
              <button 
                onClick={() => navigateToPage('pricing')} 
                className={`transition-colors ${currentPage === 'pricing' ? 'text-black' : 'text-gray-600 hover:text-black'}`}
              >
                Pricing
              </button>
              <button 
                onClick={() => navigateToPage('documentation')} 
                className={`transition-colors ${currentPage === 'documentation' ? 'text-black' : 'text-gray-600 hover:text-black'}`}
              >
                Documentation
              </button>
              <button 
                onClick={() => navigateToPage('test')}
                className="bg-red-500 text-white px-6 py-2 rounded-sm hover:bg-red-600 transition-colors mr-4"
              >
                Test
              </button>
              <button 
                onClick={() => navigateToPage('contact')}
                className="bg-black text-white px-6 py-2 rounded-sm hover:bg-gray-900 transition-colors"
              >
                Start Building
              </button>
            </nav>
          </div>
        </div>
      </header>
      
      {renderPage()}
    </div>
  );
};

export default OCXLanding;
