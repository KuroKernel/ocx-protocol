cmake_minimum_required(VERSION 3.16)
project(ocx-envoy-filter)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Include directories
include_directories(${CMAKE_CURRENT_SOURCE_DIR}/include)
include_directories(/usr/local/include)

# Source files
set(SOURCES
    src/envoy_filter_standalone.cc
)

# Create shared library
add_library(ocx_envoy_filter SHARED ${SOURCES})

# Install rules
install(TARGETS ocx_envoy_filter DESTINATION lib)
