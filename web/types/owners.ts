export interface OwnersData {
  needsLGTM: number;
  committers: string[];
  reviewers: string[];
}

export interface OwnerResponse {
  data: OwnersData;
  message: string;
}
