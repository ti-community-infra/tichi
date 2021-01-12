import { Card } from "antd";
import React from "react";
import { get } from "@/utils/request";
import {
  BASE_URL,
  COMMITTER,
  GITHUB_BASE_URL,
  REVIEWER,
} from "@/types/constant";
import { IOwnerTypeData } from "@/types/owners";

import style from "./owners.module.scss";

export default function Owners({ owners }) {
  const detailData: Partial<IOwnerTypeData> = owners.data;

  const renderMembers = (members: Array<string>, title: string) => {
    return (
      <>
        <p className={style.member}>{title}</p>
        <div className={style.wrapper}>
          {members.map((member) => (
            <Card
              className={style.container}
              cover={<img alt="pic" src={`${GITHUB_BASE_URL + member}.png`} />}
            >
              <a href={GITHUB_BASE_URL + member}>{member}</a>
            </Card>
          ))}
        </div>
      </>
    );
  };

  const reviewers = detailData.reviewers
    ? renderMembers(detailData.reviewers, REVIEWER)
    : null;
  const committers = detailData.committers
    ? renderMembers(detailData.committers, COMMITTER)
    : null;

  return (
    <>
      <p className={style.header}>needsLGTM: {detailData.needsLGTM}</p>
      {reviewers}
      {committers}
    </>
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
    props: { owners },
  };
}
