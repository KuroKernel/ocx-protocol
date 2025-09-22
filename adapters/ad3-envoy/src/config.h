#pragma once

#include <string>
#include <memory>

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace OCX {

class OCXFilterConfig {
public:
    OCXFilterConfig();
    OCXFilterConfig(const OCXFilterConfig& other);
    
    const std::string& ocxServerUrl() const;
    const std::string& apiKey() const;
    bool failClosed() const;
    
    void setOcxServerUrl(const std::string& url);
    void setApiKey(const std::string& key);
    void setFailClosed(bool fail_closed);

private:
    std::string ocx_server_url_;
    std::string api_key_;
    bool fail_closed_;
};

using OCXFilterConfigSharedPtr = std::shared_ptr<OCXFilterConfig>;

} // namespace OCX
} // namespace HttpFilters
} // namespace Extensions
} // namespace Envoy
