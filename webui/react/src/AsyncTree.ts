import { isAsyncFunction } from 'utils/data';

interface BaseNodeProps {
  title: string; // should work with the separator. no space?
}
export type Children = TreeNode[]
export type TreePath = TreeNode[]
export type TreeNode = LeafNode | NLNode;
export type ComputedChildren = (arg?: NLNode) => Children | Promise<Children>

export interface LeafNode extends BaseNodeProps {
  onAction: (arg: LeafNode) => void; // with potential response. could be shown
}

export interface NLNode extends BaseNodeProps {
  options: Children | ComputedChildren; // leaf nodes have no children
}

export const isLeafNode = (node: any): node is LeafNode =>
  node.onAction !== undefined && node.options === undefined;
export const isNLNode = (node: any): node is NLNode =>
  node.onAction === undefined && node.options !== undefined;
export const isTreeNode = (node: any): node is TreeNode =>
  node.title !== undefined && (isLeafNode(node) || isNLNode(node));

export const getNodeChildren = async (node: NLNode): Promise<Children> => {
  let children: Children = [];
  if (typeof node.options === 'function') {
    if (isAsyncFunction(node.options)) {
      children = await node.options(node);
    } else {
      children = node.options(node) as Children;
    }
  } else {
    children = node.options;
  }
  return children;
};

export const traverseTree = async (address: string[], startNode: NLNode): Promise<TreePath> => {
  let curNode: TreeNode = startNode;
  const path: TreePath = [ curNode ];
  let i = 0;
  while(isNLNode(curNode) && i<address.length) {
    const children: Children = await getNodeChildren(curNode);
    const rv = children.find(n => n.title === address[i]);
    if (rv === undefined) break;
    curNode = rv;
    i++;
    path.push(curNode);
  }
  if (i < address.length) throw new Error('bad path');
  return path;
};
