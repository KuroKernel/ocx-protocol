#include <string>
#include <memory>

#include "extensions/filters/http/common/pass_through_filter.h"
#include "envoy/http/filter.h"
#include "envoy/upstream/cluster_manager.h"
#include "source/common/http/utility.h"
#include "source/common/buffer/buffer_impl.h"

namespace Envoy {
namespace Extensions {
namespace HttpFilters {
namespace OCXFilter {

class OCXHttpFilter : public Http::PassThroughDecoderFilter,
                      public Logger::Loggable<Logger::Id::filter> {
public:
  OCXHttpFilter(const std::string& ocx_server_url, 
                const std::string& api_key,
                bool fail_closed)
      : ocx_server_url_(ocx_server_url), 
        api_key_(api_key),
        fail_closed_(fail_closed) {}

  // Decoder filter interface
  Http::FilterHeadersStatus decodeHeaders(Http::RequestHeaderMap& headers, 
                                        bool end_stream) override {
    request_headers_ = &headers;
    
    // Check if request has OCX receipt header
    const auto receipt_header = headers.get(Http::LowerCaseString("x-ocx-receipt"));
    if (receipt_header.empty()) {
      if (fail_closed_) {
        ENVOY_LOG(warn, "OCX Filter: No receipt header found, rejecting request");
        decoder_callbacks_->sendLocalReply(
          Http::Code::Unauthorized, 
          "OCX receipt required", 
          nullptr, 
          absl::nullopt, 
          "ocx_receipt_missing");
        return Http::FilterHeadersStatus::StopIteration;
      }
      return Http::FilterHeadersStatus::Continue;
    }

    if (end_stream) {
      return verifyReceipt("");
    }
    
    return Http::FilterHeadersStatus::StopIteration;
  }

  Http::FilterDataStatus decodeData(Buffer::Instance& data, bool end_stream) override {
    request_body_.add(data);
    
    if (end_stream) {
      const std::string body = request_body_.toString();
      Http::FilterHeadersStatus status = verifyReceipt(body);
      
      if (status == Http::FilterHeadersStatus::Continue) {
        return Http::FilterDataStatus::Continue;
      } else {
        return Http::FilterDataStatus::StopIterationNoBuffer;
      }
    }
    
    return Http::FilterDataStatus::StopIterationNoBuffer;
  }

private:
  Http::FilterHeadersStatus verifyReceipt(const std::string& request_body) {
    const auto start_time = std::chrono::high_resolution_clock::now();
    
    // Extract OCX receipt from header
    const auto receipt_header = request_headers_->get(Http::LowerCaseString("x-ocx-receipt"));
    if (receipt_header.empty()) {
      return handleVerificationFailure("missing_receipt");
    }

    const std::string receipt_base64 = std::string(receipt_header[0]->value().getStringView());
    
    // Create verification request
    Json::ObjectSharedPtr verification_request = std::make_shared<Json::Object>();
    verification_request->setString("receipt_data", receipt_base64);
    verification_request->setString("request_body", request_body);
    
    // Send verification request to OCX server
    Http::RequestMessagePtr request = std::make_unique<Http::RequestMessageImpl>();
    request->headers().setMethod(Http::Headers::get().MethodValues.Post);
    request->headers().setPath("/api/v1/verify");
    request->headers().setHost(getOCXServerHost());
    request->headers().setContentType("application/json");
    request->headers().setReferenceKey(Http::Headers::get().Authorization, 
                                     "Bearer " + api_key_);
    
    const std::string json_body = verification_request->asJsonString();
    request->body().add(json_body);
    request->headers().setContentLength(json_body.size());

    // Make HTTP request to OCX server
    Http::AsyncClient::Request* async_request = 
        decoder_callbacks_->clusterManager()
            .httpAsyncClientForCluster("ocx_cluster")
            .send(std::move(request), *this,
                  Http::AsyncClient::RequestOptions().setTimeout(
                      std::chrono::milliseconds(5000)));

    if (!async_request) {
      return handleVerificationFailure("ocx_server_unavailable");
    }

    // Request is async, will continue in onSuccess/onFailure
    return Http::FilterHeadersStatus::StopIteration;
  }

  void onSuccess(const Http::AsyncClient::Request&, 
                Http::ResponseMessagePtr&& response) override {
    const auto end_time = std::chrono::high_resolution_clock::now();
    const auto duration = std::chrono::duration_cast<std::chrono::microseconds>(
        end_time - start_time_).count();

    if (response->headers().getStatusValue() == "200") {
      ENVOY_LOG(debug, "OCX verification successful in {}μs", duration);
      
      // Add verification metadata to request
      request_headers_->addReferenceKey(Http::LowerCaseString("x-ocx-verified"), "true");
      request_headers_->addReferenceKey(Http::LowerCaseString("x-ocx-duration"), 
                                       std::to_string(duration));
      
      decoder_callbacks_->continueDecoding();
    } else {
      ENVOY_LOG(warn, "OCX verification failed: {}", response->headers().getStatusValue());
      handleVerificationFailure("verification_failed");
    }
  }

  void onFailure(const Http::AsyncClient::Request&, 
                Http::AsyncClient::FailureReason reason) override {
    ENVOY_LOG(error, "OCX server request failed: {}", static_cast<int>(reason));
    handleVerificationFailure("ocx_server_error");
  }

  Http::FilterHeadersStatus handleVerificationFailure(const std::string& reason) {
    if (fail_closed_) {
      decoder_callbacks_->sendLocalReply(
          Http::Code::Forbidden, 
          "OCX verification failed", 
          nullptr, 
          absl::nullopt, 
          reason);
      return Http::FilterHeadersStatus::StopIteration;
    } else {
      ENVOY_LOG(warn, "OCX verification failed ({}), allowing request", reason);
      request_headers_->addReferenceKey(Http::LowerCaseString("x-ocx-verified"), "false");
      request_headers_->addReferenceKey(Http::LowerCaseString("x-ocx-error"), reason);
      return Http::FilterHeadersStatus::Continue;
    }
  }

  std::string getOCXServerHost() {
    // Extract host from OCX server URL
    const size_t protocol_end = ocx_server_url_.find("://");
    if (protocol_end == std::string::npos) {
      return ocx_server_url_;
    }
    
    const size_t host_start = protocol_end + 3;
    const size_t path_start = ocx_server_url_.find("/", host_start);
    
    if (path_start == std::string::npos) {
      return ocx_server_url_.substr(host_start);
    }
    
    return ocx_server_url_.substr(host_start, path_start - host_start);
  }

  const std::string ocx_server_url_;
  const std::string api_key_;
  const bool fail_closed_;
  
  Http::RequestHeaderMap* request_headers_{};
  Buffer::OwnedImpl request_body_;
  std::chrono::high_resolution_clock::time_point start_time_;
};

} // namespace OCXFilter
} // namespace HttpFilters
} // namespace Extensions
} // namespace Envoy
