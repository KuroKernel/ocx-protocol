// Simplified Envoy filter that compiles without full Envoy build system
#include <string>
#include <memory>
#include <iostream>
#include <map>

// Minimal Envoy-compatible interface
namespace Envoy {
namespace Http {

enum class FilterHeadersStatus {
  Continue,
  StopIteration
};

enum class FilterDataStatus {
  Continue,
  StopIterationNoBuffer
};

class HeaderMap {
public:
  void addReferenceKey(const std::string& key, const std::string& value) {
    headers_[key] = value;
  }
  
  std::string get(const std::string& key) const {
    auto it = headers_.find(key);
    return it != headers_.end() ? it->second : "";
  }

private:
  std::map<std::string, std::string> headers_;
};

class Buffer {
public:
  void add(const std::string& data) { data_ += data; }
  std::string toString() const { return data_; }
private:
  std::string data_;
};

class StreamDecoderFilterCallbacks {
public:
  void sendLocalReply(int code, const std::string& body) {
    std::cout << "Sending reply: " << code << " " << body << std::endl;
  }
  void continueDecoding() {
    std::cout << "Continuing decoding" << std::endl;
  }
};

class PassThroughDecoderFilter {
public:
  virtual ~PassThroughDecoderFilter() = default;
  virtual FilterHeadersStatus decodeHeaders(HeaderMap& headers, bool end_stream) = 0;
  virtual FilterDataStatus decodeData(Buffer& data, bool end_stream) = 0;

protected:
  StreamDecoderFilterCallbacks* decoder_callbacks_;
};

} // namespace Http
} // namespace Envoy

// OCX Filter Implementation
class OCXHttpFilter : public Envoy::Http::PassThroughDecoderFilter {
public:
  OCXHttpFilter(const std::string& ocx_server_url, 
                const std::string& api_key,
                bool fail_closed)
      : ocx_server_url_(ocx_server_url), 
        api_key_(api_key),
        fail_closed_(fail_closed) {}

  Envoy::Http::FilterHeadersStatus decodeHeaders(Envoy::Http::HeaderMap& headers, 
                                                bool end_stream) override {
    request_headers_ = &headers;
    
    const std::string receipt_header = headers.get("x-ocx-receipt");
    if (receipt_header.empty()) {
      if (fail_closed_) {
        decoder_callbacks_->sendLocalReply(401, "OCX receipt required");
        return Envoy::Http::FilterHeadersStatus::StopIteration;
      }
      return Envoy::Http::FilterHeadersStatus::Continue;
    }

    if (end_stream) {
      return verifyReceipt("");
    }
    
    return Envoy::Http::FilterHeadersStatus::StopIteration;
  }

  Envoy::Http::FilterDataStatus decodeData(Envoy::Http::Buffer& data, 
                                          bool end_stream) override {
    request_body_.add(data.toString());
    
    if (end_stream) {
      Envoy::Http::FilterHeadersStatus status = verifyReceipt(request_body_.toString());
      return status == Envoy::Http::FilterHeadersStatus::Continue ? 
        Envoy::Http::FilterDataStatus::Continue : 
        Envoy::Http::FilterDataStatus::StopIterationNoBuffer;
    }
    
    return Envoy::Http::FilterDataStatus::StopIterationNoBuffer;
  }

private:
  Envoy::Http::FilterHeadersStatus verifyReceipt(const std::string& request_body) {
    // OCX verification logic here
    std::cout << "Verifying OCX receipt for body size: " << request_body.size() << std::endl;
    
    // For now, always succeed (implement actual verification)
    request_headers_->addReferenceKey("x-ocx-verified", "true");
    return Envoy::Http::FilterHeadersStatus::Continue;
  }

  const std::string ocx_server_url_;
  const std::string api_key_;
  const bool fail_closed_;
  
  Envoy::Http::HeaderMap* request_headers_{};
  Envoy::Http::Buffer request_body_;
};

// Factory function for Envoy integration
extern "C" {
  void* create_ocx_filter(const char* server_url, const char* api_key, bool fail_closed) {
    return new OCXHttpFilter(server_url, api_key, fail_closed);
  }
}
