#![feature(proc_macro_hygiene, decl_macro)]

#[macro_use]
extern crate rocket;
#[macro_use]
extern crate serde_derive;

use hmac::{Hmac, Mac, NewMac};
use rocket::http::Status;
use rocket::response::status;
use rocket_contrib::json::Json;
use rocket_contrib::serve::StaticFiles;
use sha1::Sha1;

// Create alias for HMAC-SHA1.
type HmacSha1 = Hmac<Sha1>;

const SHA1_PREFIX: &'static str = "sha1";

#[derive(Deserialize)]
struct Event {
    address: String,
    event: String,
    hmac: String,
    payload: String,
}

fn send_hook(
    address: &str,
    event_type: &str,
    hmac: &str,
    payload: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    let client = reqwest::blocking::Client::new();
    let resp = client
        .post(address)
        .body(payload.to_owned())
        .header("X-GitHub-Event", event_type)
        .header("X-GitHub-Delivery", "GUID")
        .header(
            "X-Hub-Signature",
            sign_payload(payload.as_bytes(), hmac.as_bytes()),
        )
        .header("Content-Type", "application/json")
        .send()?;
    if resp.status().is_success() {
        Ok(resp.text()?)
    } else {
        Ok(format!(
            "Send event success but something wrong: {}",
            resp.text()?
        ))
    }
}

fn sign_payload(payload: &[u8], key: &[u8]) -> String {
    let mut mac = HmacSha1::new_varkey(key).expect("HMAC can take key of any size");
    mac.update(payload);

    let sum = mac.finalize();

    let mut signature = SHA1_PREFIX.to_owned();
    signature.push_str(&hex::encode(sum.into_bytes()));
    signature
}

#[post("/send", data = "<event>")]
fn send(event: Json<Event>) -> status::Custom<String> {
    let result = send_hook(&event.address, &event.event, &event.hmac, &event.payload);
    match result {
        Err(e) => status::Custom(Status::InternalServerError, e.to_string()),
        Ok(result) => status::Custom(Status::Ok, result),
    }
}

fn main() {
    rocket::ignite()
        .mount("/", StaticFiles::from("static"))
        .mount("/events", routes![send])
        .launch();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sign_payload() {
        let cases = vec![
            (b"test1", b"my secret and secure key"),
            (b"test2", b"my secret and secure key"),
        ];

        for (payload, key) in cases {
            assert!(sign_payload(payload, key).contains(SHA1_PREFIX))
        }
    }
}
