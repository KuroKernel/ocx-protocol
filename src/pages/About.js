import React from 'react';
import { Users, Target, Award, Globe, ArrowRight } from 'lucide-react';

const About = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            About OCX Protocol
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            We're building the infrastructure for computational integrity. 
            Mathematical proof replaces institutional trust.
          </p>
        </div>

        {/* Mission */}
        <section className="mb-20">
          <div className="grid lg:grid-cols-2 gap-16 items-center">
            <div>
              <h2 className="text-4xl font-light tracking-tight text-black mb-8">
                Our Mission
              </h2>
              <p className="text-lg text-gray-600 leading-relaxed mb-8">
                In a world where computational results are increasingly critical to decision-making, 
                we believe that every computation should be verifiable. OCX Protocol provides the 
                cryptographic infrastructure to prove that computations happened exactly as claimed.
              </p>
              <p className="text-lg text-gray-600 leading-relaxed">
                We're eliminating the need for institutional trust by replacing it with mathematical 
                certainty. Every execution generates an immutable receipt that can be verified 
                offline, anywhere, by anyone.
              </p>
            </div>
            
            <div className="bg-gray-50 p-10 rounded-sm">
              <div className="space-y-6">
                <div className="flex items-start space-x-4">
                  <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                    <Target className="w-4 h-4 text-white" />
                  </div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-2">Deterministic Execution</h3>
                    <p className="text-gray-600">Identical results across all architectures</p>
                  </div>
                </div>
                
                <div className="flex items-start space-x-4">
                  <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                    <Award className="w-4 h-4 text-white" />
                  </div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-2">Cryptographic Proof</h3>
                    <p className="text-gray-600">Ed25519 signatures with CBOR encoding</p>
                  </div>
                </div>
                
                <div className="flex items-start space-x-4">
                  <div className="w-8 h-8 bg-black rounded-sm flex-shrink-0 mt-1 flex items-center justify-center">
                    <Globe className="w-4 h-4 text-white" />
                  </div>
                  <div>
                    <h3 className="text-xl font-medium text-black mb-2">Offline Verification</h3>
                    <p className="text-gray-600">Verify receipts without network dependency</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Team */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12 text-center">
            Our Team
          </h2>
          
          <div className="grid md:grid-cols-3 gap-12">
            <div className="text-center">
              <div className="w-24 h-24 bg-gray-200 rounded-full mx-auto mb-6 flex items-center justify-center">
                <Users className="w-12 h-12 text-gray-400" />
              </div>
              <h3 className="text-xl font-medium text-black mb-2">Engineering Team</h3>
              <p className="text-gray-600">
                Cryptography experts, systems engineers, and protocol designers 
                building the future of computational integrity.
              </p>
            </div>
            
            <div className="text-center">
              <div className="w-24 h-24 bg-gray-200 rounded-full mx-auto mb-6 flex items-center justify-center">
                <Award className="w-12 h-12 text-gray-400" />
              </div>
              <h3 className="text-xl font-medium text-black mb-2">Research Team</h3>
              <p className="text-gray-600">
                Academic researchers and protocol designers working on the 
                theoretical foundations of verifiable computation.
              </p>
            </div>
            
            <div className="text-center">
              <div className="w-24 h-24 bg-gray-200 rounded-full mx-auto mb-6 flex items-center justify-center">
                <Globe className="w-12 h-12 text-gray-400" />
              </div>
              <h3 className="text-xl font-medium text-black mb-2">Community Team</h3>
              <p className="text-gray-600">
                Developer advocates and community managers helping developers 
                integrate OCX Protocol into their applications.
              </p>
            </div>
          </div>
        </section>

        {/* Values */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12 text-center">
            Our Values
          </h2>
          
          <div className="grid lg:grid-cols-2 gap-16">
            <div className="space-y-8">
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Transparency</h3>
                <p className="text-gray-600 leading-relaxed">
                  All our code is open source. Every algorithm is documented. 
                  We believe in building trust through transparency, not obscurity.
                </p>
              </div>
              
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Security First</h3>
                <p className="text-gray-600 leading-relaxed">
                  Security is not an afterthought. Every component is designed 
                  with security as the primary consideration, from cryptography 
                  to input validation.
                </p>
              </div>
              
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Performance</h3>
                <p className="text-gray-600 leading-relaxed">
                  We optimize for real-world performance. Sub-20ms verification 
                  times and 200+ RPS throughput ensure OCX Protocol works at scale.
                </p>
              </div>
            </div>
            
            <div className="space-y-8">
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Interoperability</h3>
                <p className="text-gray-600 leading-relaxed">
                  Built on open standards like CBOR and Ed25519. Works across 
                  all platforms and programming languages. No vendor lock-in.
                </p>
              </div>
              
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Community</h3>
                <p className="text-gray-600 leading-relaxed">
                  We're building for the community, with the community. 
                  Open source, open development, open governance.
                </p>
              </div>
              
              <div>
                <h3 className="text-2xl font-medium text-black mb-4">Innovation</h3>
                <p className="text-gray-600 leading-relaxed">
                  Pushing the boundaries of what's possible with verifiable 
                  computation. Research-driven development with practical applications.
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Timeline */}
        <section className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12 text-center">
            Our Journey
          </h2>
          
          <div className="max-w-4xl mx-auto">
            <div className="space-y-12">
              <div className="flex items-start space-x-6">
                <div className="w-4 h-4 bg-black rounded-full mt-2 flex-shrink-0"></div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-2">2024 Q1 - Research Phase</h3>
                  <p className="text-gray-600">
                    Initial research into deterministic execution and cryptographic receipts. 
                    Prototype development and theoretical validation.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-6">
                <div className="w-4 h-4 bg-black rounded-full mt-2 flex-shrink-0"></div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-2">2024 Q2 - Protocol Design</h3>
                  <p className="text-gray-600">
                    Formal protocol specification. CBOR encoding, Ed25519 signatures, 
                    and cross-platform determinism design.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-6">
                <div className="w-4 h-4 bg-black rounded-full mt-2 flex-shrink-0"></div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-2">2024 Q3 - Implementation</h3>
                  <p className="text-gray-600">
                    Core implementation in Go. API server, CLI tools, and basic 
                    verification system development.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-6">
                <div className="w-4 h-4 bg-green-600 rounded-full mt-2 flex-shrink-0"></div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-2">2024 Q4 - Pilot Release</h3>
                  <p className="text-gray-600">
                    Production-ready pilot release. Enterprise features, monitoring, 
                    and comprehensive documentation.
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-6">
                <div className="w-4 h-4 bg-gray-300 rounded-full mt-2 flex-shrink-0"></div>
                <div>
                  <h3 className="text-xl font-medium text-black mb-2">2025 Q1 - Public Launch</h3>
                  <p className="text-gray-600">
                    Public launch with community features, additional SDKs, 
                    and expanded enterprise capabilities.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* CTA */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Join our mission
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Help us build the infrastructure for computational integrity. 
            Contribute to open source, join our community, or become a partner.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <a 
              href="https://github.com/ocx-protocol/ocx" 
              className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center"
            >
              Contribute on GitHub
              <ArrowRight className="w-5 h-5 ml-3" />
            </a>
            <a 
              href="https://discord.gg/ocx-protocol" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              Join Community
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default About;
