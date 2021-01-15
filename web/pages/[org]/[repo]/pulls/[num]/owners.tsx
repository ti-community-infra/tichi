import { Tag, Descriptions } from "antd";
import React from "react";
import { get } from "@/utils/request";
import { BASE_URL, GITHUB_BASE_URL } from "@/types/constant";
import { OwnersData } from "@/types/owners";

import style from "./owners.module.scss";

export default function Owners({ org, repo, num, owners }) {
  const ownersData: Partial<OwnersData> = owners.data;

  const renderMembers = (members: Array<string>) => {
    return (
      <p className={style.members}>
        {members.map((member, index) => (
          <a key={member} href={GITHUB_BASE_URL + member}>
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
            <a href={`${GITHUB_BASE_URL + org}/${repo}`}>
              {org}/{repo}
            </a>
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="PR">
          <Tag color="blue">
            <a href={`${GITHUB_BASE_URL + org}/${repo}/pull/${num}`}>
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

  let owners;
  try {
    owners = await get(`${BASE_URL}/${org}/${repo}/pulls/${num}/owners`);
  } catch (err) {
    throw err;
  }
  return {
    props: { org, repo, num, owners },
  };
}
