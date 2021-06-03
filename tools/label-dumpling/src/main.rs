use clap::Clap;
use octocrab::models;
use serde::{Deserialize, Serialize};
use std::fs::OpenOptions;

#[macro_use]
extern crate log;
#[macro_use]
extern crate dotenv_codegen;

#[derive(Clap)]
#[clap(version = "0.1.0", author = "hi-rustin <rustin.liu@gmail.com>")]
struct Opts {
    org: String,
    repo: String,
    #[clap(short, long, default_value = "label.yaml")]
    file_path: String,
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
    delete_after: Option<String>,
}

#[tokio::main]
pub async fn main() {
    pretty_env_logger::init();

    let opts: Opts = Opts::parse();
    let octocrab = octocrab::OctocrabBuilder::new()
        .personal_token(dotenv!("GITHUB_TOKEN").to_string())
        .build()
        .expect("Failed to init octocrab.");

    info!("Open the file {}.", opts.file_path);
    let file = OpenOptions::new()
        .read(true)
        .write(true)
        .create(true)
        .open(&opts.file_path)
        .unwrap_or_else(|_| panic!("Failed to open file {}.", opts.file_path));
    let mut labels: Vec<models::Label> = vec![];

    info!("Start list labels...");
    let page = octocrab
        .issues(&opts.org, &opts.repo)
        .list_labels_for_repo()
        .per_page(50)
        .send()
        .await
        .expect("Failed to list labels.");
    labels.extend_from_slice(&page.items);
    let mut next_page = page.next;
    while let Some(page) =
        (octocrab.get_page::<models::Label>(&next_page).await).expect("Failed to list labels.")
    {
        next_page = page.next;
        labels.extend_from_slice(&page.items);
    }

    let labels: Vec<Label> = labels
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
    info!("Write to file {}.", opts.file_path);
    serde_yaml::to_writer(file, &labels)
        .unwrap_or_else(|_| panic!("Failed to write file {}.", opts.file_path));
    info!("Dumping completed.");
}
