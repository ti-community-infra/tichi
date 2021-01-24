use std::convert::Infallible;

#[macro_use]
extern crate serde_derive;

use hmac::{Hmac, Mac, NewMac};
use sha1::Sha1;
use warp::{self, Filter};

// Create alias for HMAC-SHA1.
type HmacSha1 = Hmac<Sha1>;

const SHA1_PREFIX: &str = "sha1=";

#[derive(Deserialize)]
pub struct Event {
    pub address: String,
    pub event: String,
    pub hmac: String,
    pub payload: String,
}

async fn send_hook(
    address: &str,
    event_type: &str,
    hmac: &str,
    payload: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    let client = reqwest::Client::new();
    let resp = client
        .post(address)
        .body(payload.to_owned())
        .header("X-GitHub-Event", event_type)
        .header("X-GitHub-Delivery", "GUID")
        .header(
            "X-Hub-Signature",
            sign_payload(payload.as_bytes(), hmac.as_bytes()).await,
        )
        .header("Content-Type", "application/json")
        .send()
        .await?;
    if resp.status().is_success() {
        Ok(resp.text().await?)
    } else {
        Ok(format!(
            "Send event success but something wrong: {}",
            resp.text().await?
        ))
    }
}

async fn sign_payload(payload: &[u8], key: &[u8]) -> String {
    let mut mac = HmacSha1::new_varkey(key).expect("HMAC can take key of any size");
    mac.update(payload);

    let sum = mac.finalize();

    let mut signature = SHA1_PREFIX.to_owned();
    signature.push_str(&hex::encode(sum.into_bytes()));
    signature
}

pub async fn send_event(event: Event) -> Result<impl warp::Reply, Infallible> {
    let result = send_hook(&event.address, &event.event, &event.hmac, &event.payload).await;
    match result {
        Err(e) => Ok(warp::reply::json(&e.to_string())),
        Ok(result) => Ok(warp::reply::json(&result)),
    }
}

#[tokio::main]
async fn main() {
    pretty_env_logger::init();

    let log = warp::log("api::request");

    let files = warp::get()
        .and(warp::path::end())
        .and(warp::fs::dir("static"));

    let message = warp::path!("events" / "send")
        .and(warp::post())
        .and(warp::body::form())
        .and_then(send_event);

    let routes = files.or(message);

    warp::serve(routes.with(log))
        .run(([127, 0, 0, 1], 8000))
        .await;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_sign_payload() {
        let cases = vec![
            (b"test1", b"my secret and secure key"),
            (b"test2", b"my secret and secure key"),
        ];

        for (payload, key) in cases {
            assert!(sign_payload(payload, key).await.contains(SHA1_PREFIX))
        }
    }
}
