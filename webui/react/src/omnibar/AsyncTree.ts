import { isAsyncFunction } from 'utils/data';
import { Eventually } from 'utils/types';

export interface BaseNode {
  aliases?: string[];
  closeBar?: boolean;
  label?: string;
  title: string; // should work with the separator. no space?
}

export type Children = TreeNode[]
export type TreePath = TreeNode[]
export type TreeNode = LeafNode | NLNode;
export type ComputedChildren = (arg?: NLNode) => Children | Promise<Children>

export interface LeafNode extends BaseNode {
  onAction: (arg: LeafNode) => void; // with potential response. could be shown
}

// Non-leaf Node
export interface NLNode extends BaseNode {
  onCustomInput?: (input: string) => Eventually<Children>;
  options?: Children | ComputedChildren; // leaf nodes have no children
}

export const isLeafNode = (node: TreeNode): node is LeafNode =>
  'onAction' in node && !('options' in node);
export const isNLNode = (node: any): node is NLNode =>
  node.onAction === undefined && (node.options !== undefined || node.onCustomInput !== undefined);
export const isTreeNode = (node: TreeNode): node is TreeNode =>
  'title' in node && node.title !== undefined && (isLeafNode(node) || isNLNode(node));

export const getNodeChildren = async (node: TreeNode): Promise<Children> => {
  if (isLeafNode(node)) return [];
  let children: Children = [];
  if (typeof node.options === 'function') {
    if (isAsyncFunction(node.options)) {
      children = await node.options(node);
    } else {
      children = node.options(node) as Children;
    }
  } else {
    children = node.options || [];
  }
  return children;
};

/*
  Given a start node and a path: string[] get the TreePath.
*/
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

/*
  Travererse and return all staticly defined routes under a node.
*/
export const dfsStaticRoutes = (
  allRoutes: TreePath[],
  curPath: TreePath,
  node: TreeNode,
): TreePath[] => {
  curPath.push(node);
  if (isLeafNode(node)) {
    allRoutes.push(curPath);
  } else if (Array.isArray(node.options)) { // only follow statically defined children.
    node.options.forEach(child => dfsStaticRoutes(allRoutes, [ ...curPath ], child));
  } else {
    allRoutes.push(curPath);
  }
  return allRoutes;
};

// TODO add utility to check if node is child of and ancestor.
