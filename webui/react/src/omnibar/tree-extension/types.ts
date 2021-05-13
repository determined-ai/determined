import { Eventually } from 'utils/types';

export interface BaseNode {
  aliases?: string[];
  closeBar?: boolean;
  label?: string;
  title: string; // should work with the separator. no space?
}

export type Children = TreeNode[];
export type TreePath = TreeNode[];
export type TreeNode = LeafNode | NLNode;
export type ComputedChildren = (arg?: NLNode) => Eventually<Children>;
export type FinalAction = (node?: LeafNode) => Eventually<void>;

export interface LeafNode extends BaseNode {
  onAction: FinalAction;
}
// Non-leaf Node

export interface NLNode extends BaseNode {
  onCustomInput?: (input: string) => Eventually<Children>;
  options?: Children | ComputedChildren; // leaf nodes have no children
}
