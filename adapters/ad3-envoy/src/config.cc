#include "config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace OCX {

OCXFilterConfig::OCXFilterConfig() 
    : ocx_server_url_("http://localhost:8080"),
      api_key_(""),
      fail_closed_(false) {}

OCXFilterConfig::OCXFilterConfig(const OCXFilterConfig& other)
    : ocx_server_url_(other.ocx_server_url_),
      api_key_(other.api_key_),
      fail_closed_(other.fail_closed_) {}

const std::string& OCXFilterConfig::ocxServerUrl() const {
    return ocx_server_url_;
}

const std::string& OCXFilterConfig::apiKey() const {
    return api_key_;
}

bool OCXFilterConfig::failClosed() const {
    return fail_closed_;
}

void OCXFilterConfig::setOcxServerUrl(const std::string& url) {
    ocx_server_url_ = url;
}

void OCXFilterConfig::setApiKey(const std::string& key) {
    api_key_ = key;
}

void OCXFilterConfig::setFailClosed(bool fail_closed) {
    fail_closed_ = fail_closed;
}

} // namespace OCX
} // namespace HttpFilters
} // namespace Extensions
} // namespace Envoy
