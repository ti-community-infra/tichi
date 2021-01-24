use chrono::{DateTime, Utc};
use clap::Clap;
use octocrab::models;
use serde::{Deserialize, Serialize};
use std::fs::OpenOptions;

#[macro_use]
extern crate log;

#[derive(Clap)]
#[clap(version = "0.1.0", author = "hi-rustin <rustin.liu@gmail.com>")]
struct Opts {
    org: String,
    repo: String,
    #[clap(short, long)]
    token: String,
    #[clap(short, long, default_value = "label.yaml")]
    output: String,
}

#[derive(Debug, PartialEq, Serialize, Deserialize)]
struct Label {
    name: String,
    color: String,
    description: Option<String>,
    target: Option<String>,
    #[serde(rename = "prowPlugin")]
    prow_plugin: Option<String>,
    #[serde(rename = "isExternalPlugin")]
    is_external_plugin: Option<bool>,
    #[serde(rename = "addedBy")]
    added_by: Option<String>,
    previously: Option<String>,
    #[serde(rename = "deleteAfter")]
    delete_after: Option<DateTime<Utc>>,
}

#[tokio::main]
pub async fn main() {
    pretty_env_logger::init();

    let opts: Opts = Opts::parse();
    let octocrab = octocrab::OctocrabBuilder::new()
        .personal_token(opts.token)
        .build()
        .expect("Failed to build octocrab.");

    info!("Open the file {}.", opts.output);
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .open(opts.output.clone())
        .expect("Failed to open file.");
    let mut lables: Vec<models::Label> = vec![];

    info!("Start fetching labels.");
    let page = octocrab
        .issues(opts.org.clone(), opts.repo.clone())
        .list_labels_for_repo()
        .per_page(50)
        .send()
        .await
        .expect("Failed to get labels.");
    lables.extend_from_slice(&page.items);
    let mut next_page = page.next;
    while let Some(page) =
        (octocrab.get_page::<models::Label>(&next_page).await).expect("Failed to get labels.")
    {
        next_page = page.next;
        lables.extend_from_slice(&page.items);
    }

    let lables: Vec<Label> = lables
        .iter()
        .map(|l| Label {
            name: l.name.clone(),
            color: l.color.clone(),
            description: l.description.clone(),
            target: None,
            prow_plugin: None,
            is_external_plugin: None,
            added_by: None,
            previously: None,
            delete_after: None,
        })
        .collect();
    info!("Write to file {}.", opts.output);
    serde_yaml::to_writer(file, &lables).expect("Failed to write file.");
    info!("Dumping completed.");
}
