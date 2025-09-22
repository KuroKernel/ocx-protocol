use std::env;
use std::path::PathBuf;

fn main() {
    // Generate header file during build
    let out_path = PathBuf::from(env::var("OUT_DIR").unwrap());
    
    // Copy header to output directory
    std::fs::copy("ocx_verify.h", out_path.join("ocx_verify.h"))
        .expect("Failed to copy header file");
    
    println!("cargo:rerun-if-changed=src/ffi.rs");
    println!("cargo:rerun-if-changed=ocx_verify.h");
    println!("cargo:rerun-if-changed=build.rs");
}
