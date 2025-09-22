#include "envoy/http/filter.h"
#include "envoy/http/header_map.h"
#include "envoy/registry/registry.h"
#include "envoy/server/filter_config.h"

#include "common/http/utility.h"
#include "common/http/pass_through_filter.h"

#include "config.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace OCX {

class OCXFilter : public Http::PassThroughFilter {
public:
  OCXFilter(OCXFilterConfigSharedPtr config) : config_(config) {}

  Http::FilterHeadersStatus decodeHeaders(Http::RequestHeaderMap& headers, bool) override {
    // Add OCX headers to the request
    headers.addCopy("x-ocx-request-id", generateRequestId());
    headers.addCopy("x-ocx-timestamp", std::to_string(getCurrentTimestamp()));
    
    return Http::FilterHeadersStatus::Continue;
  }

  Http::FilterHeadersStatus encodeHeaders(Http::ResponseHeaderMap& headers, bool) override {
    // Add OCX headers to the response
    headers.addCopy("x-ocx-response-id", generateRequestId());
    headers.addCopy("x-ocx-verification", "enabled");
    
    return Http::FilterHeadersStatus::Continue;
  }

private:
  OCXFilterConfigSharedPtr config_;
  
  std::string generateRequestId() {
    return "ocx-" + std::to_string(getCurrentTimestamp());
  }
  
  uint64_t getCurrentTimestamp() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
  }
};

class OCXFilterFactory : public Server::Configuration::NamedHttpFilterConfigFactory {
public:
  Http::FilterFactoryCb createFilterFactoryFromProto(
      const Protobuf::Message& proto_config,
      const std::string& stats_prefix,
      Server::Configuration::FactoryContext& context) override {
    
    const auto& config = dynamic_cast<const OCXFilterConfig&>(proto_config);
    auto shared_config = std::make_shared<OCXFilterConfig>(config);
    
    return [shared_config](Http::FilterChainFactoryCallbacks& callbacks) -> void {
      callbacks.addStreamFilter(std::make_shared<OCXFilter>(shared_config));
    };
  }

  ProtobufTypes::MessagePtr createEmptyConfigProto() override {
    return std::make_unique<OCXFilterConfig>();
  }

  std::string name() const override { return "ocx_filter"; }
};

static Registry::RegisterFactory<OCXFilterFactory, Server::Configuration::NamedHttpFilterConfigFactory>
    register_;

} // namespace OCX
} // namespace HttpFilters
} // namespace Extensions
} // namespace Envoy
