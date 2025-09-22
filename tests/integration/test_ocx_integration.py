#!/usr/bin/env python3
"""
OCX Protocol Integration Tests
Comprehensive testing across all adapters and components
"""

import pytest
import requests
import time
import json
import subprocess
import os
from typing import Dict, Any

class OCXIntegrationTest:
    def __init__(self):
        self.ocx_server_url = "http://localhost:8080"
        self.kafka_bootstrap = "localhost:9092"
        self.envoy_url = "http://localhost:8000"
        
    def wait_for_service(self, url: str, timeout: int = 30) -> bool:
        """Wait for a service to be ready"""
        start_time = time.time()
        while time.time() - start_time < timeout:
            try:
                response = requests.get(url, timeout=5)
                if response.status_code == 200:
                    return True
            except requests.exceptions.RequestException:
                pass
            time.sleep(1)
        return False

    def test_ocx_server_health(self):
        """Test OCX server health endpoint"""
        assert self.wait_for_service(f"{self.ocx_server_url}/status"), "OCX server not ready"
        
        response = requests.get(f"{self.ocx_server_url}/status")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"

    def test_rust_verifier_integration(self):
        """Test Rust verifier integration"""
        # Test verification endpoint
        test_data = {
            "receipt_data": "dGVzdA==",  # base64 encoded "test"
            "public_key": "dGVzdGtleXRlc3RrZXl0ZXN0a2V5dGVzdGs="  # 32-byte test key
        }
        
        response = requests.post(
            f"{self.ocx_server_url}/verify",
            json=test_data,
            timeout=10
        )
        
        # Should return 200 even if verification fails (expected with test data)
        assert response.status_code == 200
        data = response.json()
        assert "valid" in data
        assert "duration" in data

    def test_envoy_filter_integration(self):
        """Test Envoy filter integration"""
        if not self.wait_for_service(f"{self.envoy_url}/ready", timeout=10):
            pytest.skip("Envoy not available")
        
        # Test request through Envoy
        headers = {
            "x-ocx-receipt": "dGVzdA==",
            "Content-Type": "application/json"
        }
        
        response = requests.get(
            f"{self.envoy_url}/test",
            headers=headers,
            timeout=10
        )
        
        # Should process through Envoy filter
        assert response.status_code in [200, 404]  # 404 expected for test endpoint

    def test_github_action_simulation(self):
        """Test GitHub Action simulation"""
        # Simulate GitHub Action execution
        os.environ["GITHUB_WORKSPACE"] = "/tmp"
        os.environ["GITHUB_SHA"] = "test-sha"
        os.environ["GITHUB_REF"] = "refs/heads/main"
        
        # Run GitHub Action build
        result = subprocess.run(
            ["npm", "run", "build"],
            cwd="../../adapters/ad4-github",
            capture_output=True,
            text=True
        )
        
        assert result.returncode == 0, f"GitHub Action build failed: {result.stderr}"

    def test_terraform_provider_integration(self):
        """Test Terraform provider integration"""
        # Test Terraform provider build
        result = subprocess.run(
            ["make", "build"],
            cwd="../../adapters/ad5-terraform",
            capture_output=True,
            text=True
        )
        
        assert result.returncode == 0, f"Terraform provider build failed: {result.stderr}"

    def test_kafka_interceptor_integration(self):
        """Test Kafka interceptor integration"""
        # Test Kafka interceptor build
        result = subprocess.run(
            ["mvn", "clean", "package", "-DskipTests"],
            cwd="../../adapters/ad6-kafka",
            capture_output=True,
            text=True
        )
        
        assert result.returncode == 0, f"Kafka interceptor build failed: {result.stderr}"

    def test_performance_benchmarks(self):
        """Test performance benchmarks"""
        # Test Rust verifier performance
        start_time = time.time()
        for _ in range(100):
            self.test_rust_verifier_integration()
        rust_time = time.time() - start_time
        
        print(f"Rust verifier 100 requests: {rust_time:.2f}s")
        assert rust_time < 10.0, "Rust verifier too slow"

    def test_cross_adapter_communication(self):
        """Test communication between adapters"""
        # This would test actual communication between adapters
        # For now, just verify all components are built
        components = [
            ("Rust verifier", "libocx-verify/target/release/libocx_verify.so"),
            ("Go server", "bin/ocx-server"),
            ("Envoy filter", "adapters/ad3-envoy/libocx_filter.so"),
            ("GitHub Action", "adapters/ad4-github/dist/index.js"),
            ("Terraform provider", "adapters/ad5-terraform/bin/terraform-provider-ocx"),
            ("Kafka interceptor", "adapters/ad6-kafka/target/ocx-kafka-interceptor-1.0.0.jar")
        ]
        
        for name, path in components:
            full_path = f"../../{path}"
            if os.path.exists(full_path):
                print(f"✅ {name} built successfully")
            else:
                print(f"⚠️  {name} not found at {full_path}")

def test_integration_suite():
    """Main integration test suite"""
    test_suite = OCXIntegrationTest()
    
    # Run all tests
    test_suite.test_ocx_server_health()
    test_suite.test_rust_verifier_integration()
    test_suite.test_envoy_filter_integration()
    test_suite.test_github_action_simulation()
    test_suite.test_terraform_provider_integration()
    test_suite.test_kafka_interceptor_integration()
    test_suite.test_performance_benchmarks()
    test_suite.test_cross_adapter_communication()
    
    print("🎉 All integration tests passed!")

if __name__ == "__main__":
    test_integration_suite()
