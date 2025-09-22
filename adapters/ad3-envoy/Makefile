# Envoy Filter Build System
.PHONY: build test clean benchmark

# Build configuration
CXX = g++
CXXFLAGS = -std=c++17 -Wall -Wextra -O2 -fPIC
INCLUDES = -I/usr/local/include/envoy -I/usr/include/protobuf -I/usr/include/grpc++
LIBS = -lenvoy -lprotobuf -lgrpc++ -lgrpc -lnlohmann_json

# Source files
SOURCES = envoy_filter.cc config.cc
OBJECTS = $(SOURCES:.cc=.o)
TARGET = libocx_filter.so

build: $(TARGET)

$(TARGET): $(OBJECTS)
	$(CXX) -shared -o $@ $^ $(LIBS)

%.o: %.cc
	$(CXX) $(CXXFLAGS) $(INCLUDES) -c $< -o $@

test: build
	@echo "Running Envoy filter tests..."
	./test_envoy_filter.sh

clean:
	rm -f $(OBJECTS) $(TARGET)

benchmark: build
	@echo "Running Envoy filter benchmarks..."
	./benchmark_envoy_filter.sh

install: build
	cp $(TARGET) /usr/local/lib/
	ldconfig
