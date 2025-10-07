import React from 'react';
import { Clock, Bell, Mail } from 'lucide-react';

const Status = () => {
  return (
    <div className="min-h-screen bg-white pt-20">
      <div className="max-w-4xl mx-auto px-8 py-16">
        <div className="text-center mb-16">
          <div className="inline-flex items-center justify-center w-20 h-20 bg-gray-100 rounded-full mb-8">
            <Clock className="w-10 h-10 text-gray-600" />
          </div>
          <h1 className="text-5xl font-light tracking-tight text-black mb-8">
            Status Page Coming Soon
          </h1>
          <p className="text-xl text-gray-700 leading-relaxed max-w-2xl mx-auto">
            We're preparing a comprehensive status page to keep you informed about
            the health and performance of OCX Protocol services.
          </p>
        </div>

        <div className="bg-gray-50 p-12 rounded-sm mb-12">
          <h2 className="text-2xl font-medium text-black mb-6 text-center">What to Expect</h2>
          <div className="space-y-6">
            <div className="flex items-start space-x-4">
              <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
              <div>
                <h3 className="font-medium text-black mb-2">Real-time System Status</h3>
                <p className="text-gray-700">
                  Monitor the operational status of all OCX Protocol services including
                  API servers, verification engines, and storage systems.
                </p>
              </div>
            </div>
            <div className="flex items-start space-x-4">
              <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
              <div>
                <h3 className="font-medium text-black mb-2">Performance Metrics</h3>
                <p className="text-gray-700">
                  View live performance data including API latency, throughput,
                  and verification times.
                </p>
              </div>
            </div>
            <div className="flex items-start space-x-4">
              <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
              <div>
                <h3 className="font-medium text-black mb-2">Incident History</h3>
                <p className="text-gray-700">
                  Access detailed incident reports and resolution timelines to understand
                  any service disruptions.
                </p>
              </div>
            </div>
            <div className="flex items-start space-x-4">
              <div className="w-2 h-2 bg-black rounded-full mt-2 flex-shrink-0"></div>
              <div>
                <h3 className="font-medium text-black mb-2">Uptime Statistics</h3>
                <p className="text-gray-700">
                  Review historical uptime data and availability metrics across
                  all service components.
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="border-2 border-black p-12 rounded-sm text-center">
          <Bell className="w-16 h-16 text-black mx-auto mb-6" />
          <h2 className="text-2xl font-medium text-black mb-4">
            Get Notified When We Launch
          </h2>
          <p className="text-gray-700 mb-8 max-w-xl mx-auto">
            Be the first to know when our status page goes live. We'll send you a
            single notification email when it's ready.
          </p>
          <div className="max-w-md mx-auto">
            <div className="flex space-x-4">
              <input
                type="email"
                placeholder="Enter your email"
                className="flex-1 px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
              />
              <button className="bg-black text-white px-6 py-3 rounded-sm hover:bg-gray-900 transition-colors flex items-center">
                <Mail className="w-5 h-5 mr-2" />
                Notify Me
              </button>
            </div>
            <p className="text-sm text-gray-500 mt-4">
              We'll only send you one email when the status page launches.
            </p>
          </div>
        </div>

        <div className="mt-12 text-center">
          <p className="text-gray-600 mb-4">
            In the meantime, check your service health using our API health endpoint:
          </p>
          <div className="bg-black text-green-400 p-4 rounded-sm font-mono text-sm inline-block">
            curl https://api.ocx.world/health
          </div>
        </div>
      </div>
    </div>
  );
};

export default Status;
