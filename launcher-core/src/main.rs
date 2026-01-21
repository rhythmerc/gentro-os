use axum::http::{header, Method, StatusCode};
use axum::response::IntoResponse;
use axum::routing::post;
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use std::env;
use std::fs;
use std::io;
use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use tokio::io::{AsyncReadExt, AsyncWriteExt};
use tokio::net::{TcpListener, UnixListener, UnixStream};
use tower_http::cors::CorsLayer;
use tracing::{error, info, warn};

const DEFAULT_SOCKET_PATH: &str = "/run/gentro/launcher.sock";
const DEFAULT_DATA_DIR: &str = "/data/gentro";
const DEFAULT_LOG_PATH: &str = "/data/logs/gentro-core.log";

#[derive(Debug, Serialize)]
struct JsonRpcResponse<T> {
    jsonrpc: &'static str,
    id: serde_json::Value,
    result: T,
}

#[derive(Debug, Serialize)]
struct JsonRpcError {
    code: i32,
    message: String,
}

#[derive(Debug, Serialize)]
struct JsonRpcErrorResponse {
    jsonrpc: &'static str,
    id: serde_json::Value,
    error: JsonRpcError,
}

#[derive(Debug, Deserialize)]
struct JsonRpcRequest {
    jsonrpc: String,
    id: serde_json::Value,
    method: String,
    #[serde(default)]
    params: serde_json::Value,
}

#[derive(Debug, Serialize)]
struct StatusResult {
    name: &'static str,
    version: &'static str,
    status: &'static str,
}

#[derive(Debug, Serialize)]
struct CapabilitiesResult {
    emulator: &'static str,
    settings: Vec<serde_json::Value>,
}

#[derive(Debug, Serialize)]
struct LibraryListResult {
    games: Vec<serde_json::Value>,
}

#[derive(Debug, thiserror::Error)]
enum CoreError {
    #[error("invalid json-rpc request")]
    InvalidRequest,
    #[error("io error: {0}")]
    Io(#[from] io::Error),
    #[error("database error: {0}")]
    Db(#[from] rusqlite::Error),
    #[error("serialization error: {0}")]
    Json(#[from] serde_json::Error),
}

#[tokio::main]
async fn main() -> Result<(), CoreError> {
    setup_logging()?;

    let data_dir = data_dir();
    ensure_dir(&data_dir)?;
    init_db(&data_dir)?;

    let socket_path = socket_path();
    if let Some(parent) = socket_path.parent() {
        ensure_dir(parent)?;
    }

    if socket_path.exists() {
        fs::remove_file(&socket_path)?;
    }

    let tcp_addr = tcp_addr();
    let mut unix_error = None;
    let listener = match UnixListener::bind(&socket_path) {
        Ok(listener) => {
            info!("launcher-core listening on {:?}", socket_path);
            Some(listener)
        }
        Err(err) => {
            warn!(error = %err, "failed to bind unix socket");
            unix_error = Some(err);
            None
        }
    };

    if listener.is_none() && tcp_addr.is_none() {
        return Err(CoreError::Io(unix_error.unwrap_or_else(|| {
            io::Error::new(io::ErrorKind::Other, "no listeners configured")
        })));
    }

    if let Some(addr) = tcp_addr {
        if listener.is_some() {
            tokio::spawn(async move {
                if let Err(err) = serve_http(&addr).await {
                    error!(error = %err, "http server error");
                }
            });
        } else {
            return serve_http(&addr).await;
        }
    }

    if let Some(listener) = listener {
        loop {
            let (stream, _) = listener.accept().await?;
            tokio::spawn(async move {
                if let Err(err) = handle_connection(stream).await {
                    error!(error = %err, "connection error");
                }
            });
        }
    }

    Ok(())
}

fn setup_logging() -> Result<(), CoreError> {
    let log_path = env::var("GENTRO_LOG_PATH").unwrap_or_else(|_| DEFAULT_LOG_PATH.to_string());
    if let Some(parent) = Path::new(&log_path).parent() {
        let _ = fs::create_dir_all(parent);
    }

    let make_writer = move || {
        fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(&log_path)
            .map(|file| Box::new(file) as Box<dyn io::Write + Send>)
            .unwrap_or_else(|_| Box::new(io::stdout()) as Box<dyn io::Write + Send>)
    };

    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .with_writer(make_writer)
        .init();

    Ok(())
}

fn socket_path() -> PathBuf {
    env::var("GENTRO_SOCKET_PATH")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from(DEFAULT_SOCKET_PATH))
}

fn tcp_addr() -> Option<String> {
    env::var("GENTRO_TCP_ADDR").ok()
}

fn data_dir() -> PathBuf {
    env::var("GENTRO_DATA_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|_| PathBuf::from(DEFAULT_DATA_DIR))
}

fn ensure_dir(path: &Path) -> Result<(), CoreError> {
    fs::create_dir_all(path)?;
    Ok(())
}

fn init_db(data_dir: &Path) -> Result<(), CoreError> {
    let db_path = data_dir.join("core.db");
    let conn = rusqlite::Connection::open(db_path)?;
    conn.execute_batch(
        "
        CREATE TABLE IF NOT EXISTS emulators (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            system TEXT NOT NULL,
            command TEXT NOT NULL,
            args TEXT
        );
        CREATE TABLE IF NOT EXISTS games (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            platform TEXT NOT NULL,
            source TEXT NOT NULL,
            install_state TEXT NOT NULL,
            launch_target TEXT
        );
        CREATE TABLE IF NOT EXISTS game_overrides (
            game_id TEXT NOT NULL,
            emulator_id TEXT NOT NULL,
            key TEXT NOT NULL,
            value TEXT NOT NULL,
            PRIMARY KEY (game_id, emulator_id, key)
        );
        ",
    )?;
    Ok(())
}

async fn handle_connection(mut stream: UnixStream) -> Result<(), CoreError> {
    let mut buffer = Vec::new();
    stream.read_to_end(&mut buffer).await?;
    if buffer.is_empty() {
        return Err(CoreError::InvalidRequest);
    }

    let request: JsonRpcRequest = serde_json::from_slice(&buffer)?;
    let response = handle_request(&request)?;
    let body = serde_json::to_vec(&response)?;
    stream.write_all(&body).await?;
    Ok(())
}

fn handle_request(request: &JsonRpcRequest) -> Result<serde_json::Value, CoreError> {
    if request.jsonrpc != "2.0" {
        return Err(CoreError::InvalidRequest);
    }

    let _params = &request.params;

    match request.method.as_str() {
        "core.status" => {
            let result = StatusResult {
                name: "gentro-core",
                version: env!("CARGO_PKG_VERSION"),
                status: "ok",
            };
            Ok(serde_json::to_value(JsonRpcResponse {
                jsonrpc: "2.0",
                id: request.id.clone(),
                result,
            })?)
        }
        "emulator.capabilities" => {
            let result = CapabilitiesResult {
                emulator: "dolphin",
                settings: Vec::new(),
            };
            Ok(serde_json::to_value(JsonRpcResponse {
                jsonrpc: "2.0",
                id: request.id.clone(),
                result,
            })?)
        }
        "library.list" => {
            let result = LibraryListResult { games: Vec::new() };
            Ok(serde_json::to_value(JsonRpcResponse {
                jsonrpc: "2.0",
                id: request.id.clone(),
                result,
            })?)
        }
        _ => Ok(serde_json::to_value(JsonRpcErrorResponse {
            jsonrpc: "2.0",
            id: request.id.clone(),
            error: JsonRpcError {
                code: -32601,
                message: "Method not found".to_string(),
            },
        })?),
    }
}

async fn serve_http(addr: &str) -> Result<(), CoreError> {
    let allowed_origins = [
        "http://localhost:5173".parse().unwrap(),
        "http://localhost:1420".parse().unwrap(),
        "tauri://localhost".parse().unwrap(),
    ];

    let cors = CorsLayer::new()
        .allow_origin(allowed_origins)
        .allow_methods([Method::POST, Method::OPTIONS])
        .allow_headers([header::CONTENT_TYPE]);

    let router = Router::new().route("/rpc", post(handle_http)).layer(cors);
    let socket_addr: SocketAddr = addr.parse().map_err(|_| CoreError::InvalidRequest)?;
    let listener = TcpListener::bind(socket_addr).await?;
    info!("launcher-core http listening on {}", addr);
    axum::serve(listener, router)
        .await
        .map_err(|err| CoreError::Io(io::Error::new(io::ErrorKind::Other, err)))
}

async fn handle_http(Json(payload): Json<serde_json::Value>) -> impl IntoResponse {
    let request: Result<JsonRpcRequest, _> = serde_json::from_value(payload);
    let response = match request {
        Ok(value) => match handle_request(&value) {
            Ok(result) => (StatusCode::OK, Json(result)),
            Err(err) => (StatusCode::OK, Json(error_response(value.id.clone(), err))),
        },
        Err(_) => {
            let error = error_response(serde_json::Value::Null, CoreError::InvalidRequest);
            (StatusCode::BAD_REQUEST, Json(error))
        }
    };

    response.into_response()
}

fn error_response(id: serde_json::Value, err: CoreError) -> serde_json::Value {
    let (code, message) = match err {
        CoreError::InvalidRequest => (-32600, "Invalid Request".to_string()),
        CoreError::Json(error) => (-32700, format!("Parse error: {}", error)),
        CoreError::Io(error) => (-32000, format!("IO error: {}", error)),
        CoreError::Db(error) => (-32001, format!("DB error: {}", error)),
    };

    serde_json::to_value(JsonRpcErrorResponse {
        jsonrpc: "2.0",
        id,
        error: JsonRpcError { code, message },
    })
    .unwrap_or_else(|_| serde_json::json!({"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"Internal error"}}))
}
