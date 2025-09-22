import React, { useState } from 'react';
import { Check, ArrowRight, Star, Zap, Shield, Users } from 'lucide-react';

const Pricing = () => {
  const [billingCycle, setBillingCycle] = useState('monthly');

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            Transparent Pricing
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Choose the plan that fits your needs. Scale from development to enterprise 
            with predictable, transparent pricing.
          </p>
        </div>

        {/* Billing Toggle */}
        <div className="flex justify-center mb-16">
          <div className="bg-gray-100 rounded-sm p-1 flex">
            <button
              onClick={() => setBillingCycle('monthly')}
              className={`px-6 py-2 rounded-sm text-sm font-medium transition-colors ${
                billingCycle === 'monthly'
                  ? 'bg-white text-black shadow-sm'
                  : 'text-gray-600 hover:text-black'
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setBillingCycle('yearly')}
              className={`px-6 py-2 rounded-sm text-sm font-medium transition-colors ${
                billingCycle === 'yearly'
                  ? 'bg-white text-black shadow-sm'
                  : 'text-gray-600 hover:text-black'
              }`}
            >
              Yearly
              <span className="ml-2 bg-green-100 text-green-800 px-2 py-1 rounded text-xs">Save 20%</span>
            </button>
          </div>
        </div>

        {/* Pricing Cards */}
        <div className="grid lg:grid-cols-3 gap-8 max-w-6xl mx-auto mb-20">
          {/* Development */}
          <div className="border border-gray-200 rounded-sm p-10 relative">
            <div className="text-center mb-8">
              <h3 className="text-2xl font-medium text-black mb-4">Development</h3>
              <div className="text-5xl font-light text-black mb-2">Free</div>
              <p className="text-gray-600">Perfect for getting started</p>
            </div>
            
            <ul className="space-y-4 mb-10">
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                1M verifications/month
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                100k receipts stored
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Community support
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Basic monitoring
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                OpenAPI access
              </li>
            </ul>
            
            <button className="w-full border border-black text-black py-4 rounded-sm hover:bg-black hover:text-white transition-colors font-medium">
              Start Building
            </button>
          </div>

          {/* Professional */}
          <div className="border-2 border-black rounded-sm p-10 relative bg-white">
            <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
              <span className="bg-black text-white px-6 py-2 rounded-sm text-sm font-medium flex items-center">
                <Star className="w-4 h-4 mr-2" />
                RECOMMENDED
              </span>
            </div>
            
            <div className="text-center mb-8">
              <h3 className="text-2xl font-medium text-black mb-4">Professional</h3>
              <div className="text-5xl font-light text-black mb-2">
                ${billingCycle === 'yearly' ? '239' : '299'}
                <span className="text-xl text-gray-600">/{billingCycle === 'yearly' ? 'year' : 'mo'}</span>
              </div>
              <p className="text-gray-600">For growing teams</p>
            </div>
            
            <ul className="space-y-4 mb-10">
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                20M verifications/month
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                2M receipts stored
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                99.9% SLA guarantee
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Priority support
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Advanced monitoring
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Webhook integrations
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Custom domains
              </li>
            </ul>
            
            <button className="w-full bg-black text-white py-4 rounded-sm hover:bg-gray-900 transition-colors font-medium">
              Start Trial
            </button>
          </div>

          {/* Enterprise */}
          <div className="border border-gray-200 rounded-sm p-10 relative">
            <div className="text-center mb-8">
              <h3 className="text-2xl font-medium text-black mb-4">Enterprise</h3>
              <div className="text-5xl font-light text-black mb-2">Custom</div>
              <p className="text-gray-600">For large organizations</p>
            </div>
            
            <ul className="space-y-4 mb-10">
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Unlimited verifications
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                On-premises deployment
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                SSO integration
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Dedicated support
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Custom SLA
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                White-label options
              </li>
              <li className="flex items-center text-gray-600">
                <Check className="w-5 h-5 text-black mr-4 flex-shrink-0" />
                Compliance support
              </li>
            </ul>
            
            <button className="w-full border border-black text-black py-4 rounded-sm hover:bg-black hover:text-white transition-colors font-medium">
              Contact Sales
            </button>
          </div>
        </div>

        {/* Feature Comparison */}
        <div className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12 text-center">
            Feature Comparison
          </h2>
          
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-4 px-6 font-medium text-black">Features</th>
                  <th className="text-center py-4 px-6 font-medium text-black">Development</th>
                  <th className="text-center py-4 px-6 font-medium text-black">Professional</th>
                  <th className="text-center py-4 px-6 font-medium text-black">Enterprise</th>
                </tr>
              </thead>
              <tbody className="text-sm">
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">Verifications/month</td>
                  <td className="py-4 px-6 text-center">1M</td>
                  <td className="py-4 px-6 text-center">20M</td>
                  <td className="py-4 px-6 text-center">Unlimited</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">Receipts stored</td>
                  <td className="py-4 px-6 text-center">100k</td>
                  <td className="py-4 px-6 text-center">2M</td>
                  <td className="py-4 px-6 text-center">Unlimited</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">SLA</td>
                  <td className="py-4 px-6 text-center">-</td>
                  <td className="py-4 px-6 text-center">99.9%</td>
                  <td className="py-4 px-6 text-center">Custom</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">Support</td>
                  <td className="py-4 px-6 text-center">Community</td>
                  <td className="py-4 px-6 text-center">Priority</td>
                  <td className="py-4 px-6 text-center">Dedicated</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">Monitoring</td>
                  <td className="py-4 px-6 text-center">Basic</td>
                  <td className="py-4 px-6 text-center">Advanced</td>
                  <td className="py-4 px-6 text-center">Custom</td>
                </tr>
                <tr className="border-b border-gray-100">
                  <td className="py-4 px-6 text-gray-600">Integrations</td>
                  <td className="py-4 px-6 text-center">API only</td>
                  <td className="py-4 px-6 text-center">Webhooks</td>
                  <td className="py-4 px-6 text-center">Custom</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Additional Pricing Info */}
        <div className="grid lg:grid-cols-2 gap-16 mb-20">
          <div>
            <h3 className="text-2xl font-medium text-black mb-6">Overage Pricing</h3>
            <div className="space-y-4">
              <div className="flex justify-between items-center py-3 border-b border-gray-100">
                <span className="text-gray-600">Additional verifications</span>
                <span className="font-medium text-black">$10 per 1M</span>
              </div>
              <div className="flex justify-between items-center py-3 border-b border-gray-100">
                <span className="text-gray-600">Additional storage</span>
                <span className="font-medium text-black">$5 per 100k receipts</span>
              </div>
              <div className="flex justify-between items-center py-3 border-b border-gray-100">
                <span className="text-gray-600">Priority support</span>
                <span className="font-medium text-black">$500/month</span>
              </div>
            </div>
          </div>
          
          <div>
            <h3 className="text-2xl font-medium text-black mb-6">Enterprise Add-ons</h3>
            <div className="space-y-4">
              <div className="flex items-start space-x-3">
                <Shield className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                <div>
                  <span className="font-medium text-black">Compliance Support</span>
                  <p className="text-gray-600 text-sm">SOC 2, GDPR, HIPAA compliance assistance</p>
                </div>
              </div>
              <div className="flex items-start space-x-3">
                <Users className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                <div>
                  <span className="font-medium text-black">Dedicated Support</span>
                  <p className="text-gray-600 text-sm">24/7 dedicated support team</p>
                </div>
              </div>
              <div className="flex items-start space-x-3">
                <Zap className="w-5 h-5 text-black mt-1 flex-shrink-0" />
                <div>
                  <span className="font-medium text-black">Custom SLA</span>
                  <p className="text-gray-600 text-sm">Tailored service level agreements</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* FAQ */}
        <div className="mb-20">
          <h2 className="text-4xl font-light tracking-tight text-black mb-12 text-center">
            Frequently Asked Questions
          </h2>
          
          <div className="max-w-3xl mx-auto space-y-8">
            <div>
              <h3 className="text-lg font-medium text-black mb-3">What counts as a verification?</h3>
              <p className="text-gray-600">
                A verification is each time you call the /api/v1/verify endpoint to validate a receipt. 
                This includes both successful and failed verification attempts.
              </p>
            </div>
            
            <div>
              <h3 className="text-lg font-medium text-black mb-3">How is storage calculated?</h3>
              <p className="text-gray-600">
                Storage is calculated based on the number of receipts stored in your account. 
                Each receipt typically ranges from 200-500 bytes depending on the complexity of the computation.
              </p>
            </div>
            
            <div>
              <h3 className="text-lg font-medium text-black mb-3">Can I change plans anytime?</h3>
              <p className="text-gray-600">
                Yes, you can upgrade or downgrade your plan at any time. Changes take effect immediately, 
                and we'll prorate any billing differences.
              </p>
            </div>
            
            <div>
              <h3 className="text-lg font-medium text-black mb-3">What happens if I exceed my limits?</h3>
              <p className="text-gray-600">
                We'll notify you when you're approaching your limits. If you exceed them, 
                we'll charge overage fees at the rates shown above. You can also upgrade your plan to avoid overages.
              </p>
            </div>
          </div>
        </div>

        {/* CTA */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Ready to get started?
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Start with our free development plan and scale as you grow. 
            No credit card required to begin.
          </p>
          <div className="flex items-center justify-center space-x-8">
            <button className="bg-black text-white px-10 py-4 rounded-sm hover:bg-gray-900 transition-colors text-lg flex items-center">
              Start Building
              <ArrowRight className="w-5 h-5 ml-3" />
            </button>
            <a 
              href="#contact" 
              className="text-gray-600 hover:text-black transition-colors border-b border-gray-300 pb-1 text-lg"
            >
              Contact Sales
            </a>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Pricing;
