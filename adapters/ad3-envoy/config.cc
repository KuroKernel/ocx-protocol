class OCXHttpFilterConfig : public Extensions::HttpFilters::Common::FactoryBase<
                              envoy::extensions::filters::http::ocx::v3::OCXFilter> {
public:
  OCXHttpFilterConfig() : FactoryBase("envoy.filters.http.ocx") {}

private:
  Http::FilterFactoryCb createFilterFactoryFromProtoTyped(
      const envoy::extensions::filters::http::ocx::v3::OCXFilter& config,
      const std::string&,
      Server::Configuration::FactoryContext&) override {

    return [config](Http::FilterChainFactoryCallbacks& callbacks) -> void {
      callbacks.addStreamDecoderFilter(
          std::make_shared<OCXHttpFilter>(
              config.ocx_server_url(),
              config.api_key(),
              config.fail_closed()));
    };
  }
};
