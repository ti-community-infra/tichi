export interface IOwnerTypeData {
  needsLGTM: number;
  committers: string[];
  reviewers: string[];
}

export interface OwnerType {
  data: IOwnerTypeData;
  message: string;
}
