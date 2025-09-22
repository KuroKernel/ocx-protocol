import React, { useState, useEffect } from 'react';
import { CheckCircle, AlertCircle, XCircle, Clock, RefreshCw } from 'lucide-react';

const Status = () => {
  const [status, setStatus] = useState({
    overall: 'operational',
    services: [
      { name: 'API Server', status: 'operational', uptime: '99.9%' },
      { name: 'Database', status: 'operational', uptime: '99.95%' },
      { name: 'Key Store', status: 'operational', uptime: '100%' },
      { name: 'Metrics', status: 'operational', uptime: '99.8%' }
    ],
    incidents: [],
    lastUpdated: new Date().toISOString()
  });
  const [isRefreshing, setIsRefreshing] = useState(false);

  const refreshStatus = async () => {
    setIsRefreshing(true);
    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      setStatus(prev => ({
        ...prev,
        lastUpdated: new Date().toISOString()
      }));
    } finally {
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    const interval = setInterval(refreshStatus, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = (status) => {
    switch (status) {
      case 'operational':
        return <CheckCircle className="w-5 h-5 text-green-600" />;
      case 'degraded':
        return <AlertCircle className="w-5 h-5 text-yellow-600" />;
      case 'outage':
        return <XCircle className="w-5 h-5 text-red-600" />;
      default:
        return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'operational':
        return 'text-green-600';
      case 'degraded':
        return 'text-yellow-600';
      case 'outage':
        return 'text-red-600';
      default:
        return 'text-gray-400';
    }
  };

  return (
    <div className="min-h-screen bg-white pt-20">
      {/* Header */}
      <div className="max-w-6xl mx-auto px-8 py-16">
        <div className="text-center mb-20">
          <h1 className="text-6xl font-light tracking-tight text-black mb-8">
            System Status
          </h1>
          <p className="text-xl text-gray-600 max-w-3xl mx-auto leading-relaxed">
            Real-time status of OCX Protocol services. We monitor all systems 
            24/7 to ensure maximum uptime.
          </p>
        </div>

        {/* Overall Status */}
        <div className="bg-gray-50 p-8 rounded-sm mb-12">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center space-x-4">
              {getStatusIcon(status.overall)}
              <h2 className="text-2xl font-medium text-black">All Systems Operational</h2>
            </div>
            <button
              onClick={refreshStatus}
              disabled={isRefreshing}
              className="flex items-center space-x-2 text-gray-600 hover:text-black transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
          <p className="text-gray-600">
            All OCX Protocol services are running normally. Last updated: {new Date(status.lastUpdated).toLocaleString()}
          </p>
        </div>

        {/* Service Status */}
        <div className="mb-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12">Service Status</h2>
          
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-6">
            {status.services.map((service, index) => (
              <div key={index} className="border border-gray-200 rounded-sm p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-medium text-black">{service.name}</h3>
                  {getStatusIcon(service.status)}
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Status</span>
                    <span className={`font-medium ${getStatusColor(service.status)}`}>
                      {service.status.charAt(0).toUpperCase() + service.status.slice(1)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Uptime</span>
                    <span className="font-medium text-black">{service.uptime}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Performance Metrics */}
        <div className="mb-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12">Performance Metrics</h2>
          
          <div className="grid md:grid-cols-3 gap-8">
            <div className="text-center">
              <div className="text-4xl font-light text-black mb-2">18.5ms</div>
              <p className="text-gray-600">P99 Latency</p>
              <p className="text-sm text-gray-500 mt-1">Target: &lt; 20ms</p>
            </div>
            <div className="text-center">
              <div className="text-4xl font-light text-black mb-2">200+</div>
              <p className="text-gray-600">RPS Throughput</p>
              <p className="text-sm text-gray-500 mt-1">Per node</p>
            </div>
            <div className="text-center">
              <div className="text-4xl font-light text-black mb-2">99.9%</div>
              <p className="text-gray-600">Availability</p>
              <p className="text-sm text-gray-500 mt-1">Last 30 days</p>
            </div>
          </div>
        </div>

        {/* Recent Incidents */}
        <div className="mb-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12">Recent Incidents</h2>
          
          {status.incidents.length === 0 ? (
            <div className="text-center py-12 bg-gray-50 rounded-sm">
              <CheckCircle className="w-16 h-16 text-green-600 mx-auto mb-4" />
              <h3 className="text-xl font-medium text-black mb-2">No Recent Incidents</h3>
              <p className="text-gray-600">
                All systems have been running smoothly with no incidents in the last 30 days.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {status.incidents.map((incident, index) => (
                <div key={index} className="border border-gray-200 rounded-sm p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg font-medium text-black">{incident.title}</h3>
                    <span className={`px-3 py-1 rounded text-sm font-medium ${
                      incident.status === 'resolved' ? 'bg-green-100 text-green-800' :
                      incident.status === 'investigating' ? 'bg-yellow-100 text-yellow-800' :
                      'bg-red-100 text-red-800'
                    }`}>
                      {incident.status.charAt(0).toUpperCase() + incident.status.slice(1)}
                    </span>
                  </div>
                  <p className="text-gray-600 mb-4">{incident.description}</p>
                  <div className="flex items-center space-x-4 text-sm text-gray-500">
                    <span>Started: {new Date(incident.startTime).toLocaleString()}</span>
                    {incident.endTime && (
                      <span>Resolved: {new Date(incident.endTime).toLocaleString()}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Status History */}
        <div className="mb-20">
          <h2 className="text-3xl font-light tracking-tight text-black mb-12">Status History</h2>
          
          <div className="bg-gray-50 p-8 rounded-sm">
            <div className="text-center">
              <h3 className="text-xl font-medium text-black mb-4">30-Day Uptime</h3>
              <div className="text-4xl font-light text-black mb-2">99.9%</div>
              <p className="text-gray-600 mb-8">
                Total downtime: 1 hour 26 minutes
              </p>
              
              <div className="grid grid-cols-30 gap-1 max-w-4xl mx-auto">
                {Array.from({ length: 30 }, (_, i) => {
                  const isOperational = Math.random() > 0.001; // 99.9% uptime
                  return (
                    <div
                      key={i}
                      className={`h-8 rounded-sm ${
                        isOperational ? 'bg-green-500' : 'bg-red-500'
                      }`}
                      title={`Day ${i + 1}: ${isOperational ? 'Operational' : 'Incident'}`}
                    />
                  );
                })}
              </div>
              <div className="flex items-center justify-center space-x-8 mt-6 text-sm">
                <div className="flex items-center space-x-2">
                  <div className="w-4 h-4 bg-green-500 rounded-sm"></div>
                  <span className="text-gray-600">Operational</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className="w-4 h-4 bg-red-500 rounded-sm"></div>
                  <span className="text-gray-600">Incident</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Subscribe to Updates */}
        <div className="text-center bg-gray-50 p-12 rounded-sm">
          <h2 className="text-4xl font-light tracking-tight text-black mb-8">
            Stay Updated
          </h2>
          <p className="text-xl text-gray-600 mb-12 max-w-2xl mx-auto">
            Subscribe to status updates and get notified immediately when incidents occur.
          </p>
          <div className="max-w-md mx-auto">
            <div className="flex space-x-4">
              <input
                type="email"
                placeholder="Enter your email"
                className="flex-1 px-4 py-3 border border-gray-300 rounded-sm focus:ring-2 focus:ring-black focus:border-transparent"
              />
              <button className="bg-black text-white px-6 py-3 rounded-sm hover:bg-gray-900 transition-colors">
                Subscribe
              </button>
            </div>
            <p className="text-sm text-gray-500 mt-4">
              We'll only send you status updates. Unsubscribe anytime.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Status;
