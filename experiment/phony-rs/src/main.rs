#![feature(proc_macro_hygiene, decl_macro)]

#[macro_use]
extern crate rocket;

use hmac::{Hmac, Mac, NewMac};
use rocket::request::Form;
use sha1::Sha1;

// Create alias for HMAC-SHA1.
type HmacSha1 = Hmac<Sha1>;

#[derive(FromForm)]
struct Event {
    address: String,
    event: String,
    hmac: String,
    payload: String,
}

#[post("/send", data = "<event>")]
fn send(event: Form<Event>) -> Result<String, Box<dyn std::error::Error>> {
    send_hook(&event.address, &event.event, &event.hmac, &event.payload)
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
            "Send event success but got a err from response: {}",
            resp.text()?
        ))
    }
}

fn sign_payload(payload: &[u8], key: &[u8]) -> String {
    let mut mac = HmacSha1::new_varkey(key).expect("HMAC can take key of any size");
    mac.update(payload);

    let sum = mac.finalize();

    let mut signature = "sha1=".to_owned();
    signature.push_str(&hex::encode(sum.into_bytes()));
    signature
}

fn main() {
    rocket::ignite().mount("/events", routes![send]).launch();
}
