import { Tag, Descriptions } from "antd";
import React from "react";
import { get } from "@/utils/request";
import {
  DEFAULT_GITHUB_ENDPOINT,
  LGTM_CONFIGURATION_KEY,
  PULL_OWNERS_ENDPOINT_KEY,
  REPOS_KEY,
} from "@/types/constant";
import { OwnersData } from "@/types/owners";

import style from "./owners.module.scss";

const yaml = require("js-yaml");
const fs = require("fs");

export default function Owners({ org, repo, num, owners }) {
  const ownersData: Partial<OwnersData> = owners.data;

  const renderMembers = (members: Array<string>) => {
    return (
      <p className={style.members}>
        {members.map((member) => (
          <a key={member} href={DEFAULT_GITHUB_ENDPOINT + member}>
            {` @${member} `}
          </a>
        ))}
      </p>
    );
  };

  return (
    <div className={style.desc}>
      <Descriptions title="PR Owners" bordered>
        <Descriptions.Item label="Repo">
          <Tag color="green">
            <a href={`${DEFAULT_GITHUB_ENDPOINT + org}/${repo}`}>
              {org}/{repo}
            </a>
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="PR">
          <Tag color="blue">
            <a href={`${DEFAULT_GITHUB_ENDPOINT + org}/${repo}/pull/${num}`}>
              {org}/{repo}#{num}
            </a>
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Required LGTM Number">
          <Tag color="red">{ownersData.needsLGTM}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Committers" span={3}>
          {renderMembers(ownersData.committers)}
        </Descriptions.Item>
        <Descriptions.Item label="Reviewers" span={3}>
          {renderMembers(ownersData.reviewers)}
        </Descriptions.Item>
      </Descriptions>
    </div>
  );
}

export async function getServerSideProps(ctx) {
  const { query } = ctx;
  const { num, org, repo } = query;

  const ownersEndpoint = findOwnersEndpoint(
    process.env["EXTERNAL_PLUGINS_CONFIG"],
    org,
    repo
  );

  if (ownersEndpoint === null) {
    throw new Error(`Can not find the owners endpoint of the ${org}/${repo}.`);
  }

  let owners;
  try {
    owners = await get(
      `${ownersEndpoint}/repos/${org}/${repo}/pulls/${num}/owners`
    );
  } catch (err) {
    throw err;
  }

  return {
    props: { org, repo, num, owners },
  };
}

/**
 * Find owners endpoint of the repo.
 * @param configFilePath The external config file path.
 * @param org Repo's org.
 * @param repo Repo's name.
 */
function findOwnersEndpoint(
  configFilePath: string,
  org: string,
  repo: string
): string | null {
  let ownersURL = null;
  const externalConfig = yaml.load(fs.readFileSync(configFilePath));
  const lgtmConfigs = externalConfig[LGTM_CONFIGURATION_KEY];

  lgtmConfigs.forEach((lgtm) => {
    const repos = lgtm[REPOS_KEY];
    // Use org name or repo full name.
    repos.forEach((r) => {
      if (r === org || r === `${org}/${repo}`) {
        ownersURL = lgtm[PULL_OWNERS_ENDPOINT_KEY];
      }
    });
  });

  return ownersURL;
}
