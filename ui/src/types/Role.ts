import { IFilterable } from './Selectable';

export interface IRole {
  id: string;
  key: string;
  name: string;
  description: string;
  protected: boolean;
}

export type FilterableRole = IRole & IFilterable;
