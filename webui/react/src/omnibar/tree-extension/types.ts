import { Eventually } from 'utils/types';

export interface BaseNode {
  aliases?: string[];
  closeBar?: boolean;
  label?: string;
  title: string; // should work with the separator. no space?
}

export interface LeafNode extends BaseNode {
  onAction: FinalAction;
}

export interface NonLeafNode extends BaseNode {
  onCustomInput?: (input: string) => Eventually<Children>;
  options?: Children | ComputedChildren; // leaf nodes have no children
}

export type Children = TreeNode[];
export type TreePath = TreeNode[];
export type TreeNode = LeafNode | NonLeafNode;
export type ComputedChildren = (arg?: NonLeafNode) => Eventually<Children>;
export type FinalAction = (node?: LeafNode) => Eventually<void>;
