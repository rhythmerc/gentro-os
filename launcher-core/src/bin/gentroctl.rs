use serde_json::{json, Value};
use std::env;
use std::io::{self, Write};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::UnixStream;

const DEFAULT_SOCKET_PATH: &str = "/run/gentro/launcher.sock";

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args = Args::parse()?;
    let socket_path = args
        .socket
        .or_else(|| env::var("GENTRO_SOCKET_PATH").ok())
        .unwrap_or_else(|| DEFAULT_SOCKET_PATH.to_string());

    let request = json!({
        "jsonrpc": "2.0",
        "id": args.id,
        "method": args.method,
        "params": args.params
    });

    let mut stream = UnixStream::connect(socket_path).await?;
    let payload = serde_json::to_vec(&request)?;
    stream.write_all(&payload).await?;
    stream.shutdown().await?;

    let mut response = Vec::new();
    stream.read_to_end(&mut response).await?;
    if response.is_empty() {
        return Err("empty response".into());
    }

    let output: Value = serde_json::from_slice(&response)?;
    println!("{}", serde_json::to_string_pretty(&output)?);
    Ok(())
}

struct Args {
    socket: Option<String>,
    method: String,
    params: Value,
    id: Value,
}

impl Args {
    fn parse() -> Result<Self, Box<dyn std::error::Error>> {
        let mut socket = None;
        let mut method = None;
        let mut params = None;
        let mut id = None;

        let mut args = env::args().skip(1);
        while let Some(arg) = args.next() {
            match arg.as_str() {
                "-s" | "--socket" => {
                    socket = args.next();
                }
                "-p" | "--params" => {
                    params = args.next();
                }
                "--id" => {
                    id = args.next();
                }
                "-h" | "--help" => {
                    print_usage();
                    std::process::exit(0);
                }
                value if method.is_none() => {
                    method = Some(value.to_string());
                }
                _ => {}
            }
        }

        let method = match method {
            Some(value) => value,
            None => {
                print_usage();
                return Err("missing method".into());
            }
        };

        let params = match params {
            Some(raw) => serde_json::from_str(&raw)?,
            None => json!({}),
        };

        let id = match id {
            Some(raw) => serde_json::from_str(&raw).unwrap_or_else(|_| Value::String(raw)),
            None => json!(1),
        };

        Ok(Self {
            socket,
            method,
            params,
            id,
        })
    }
}

fn print_usage() {
    let usage = "Usage: gentroctl [options] <method>\n\
\n\
Options:\n\
  -s, --socket <path>   Socket path (default: /run/gentro/launcher.sock)\n\
  -p, --params <json>   JSON params object (default: {})\n\
  --id <json>           JSON id value (default: 1)\n\
  -h, --help            Show this help\n\
\n\
Example:\n\
  gentroctl core.status\n\
  gentroctl -p '{\"key\":\"gfx.internal_resolution\",\"value\":3}' emulator.set";
    let _ = writeln!(io::stderr(), "{usage}");
}
