import { isAsyncFunction } from 'utils/data';

import { BaseNode, Children, LeafNode, NonLeafNode, TreeNode, TreePath } from './types';

export const isBaseNode = (obj: unknown): obj is BaseNode =>
  obj instanceof Object && 'title' in obj;
export const isLeafNode = (obj: unknown): obj is LeafNode =>
  isBaseNode(obj) && 'onAction' in obj && !('options' in obj);
export const isNLNode = (obj: unknown): obj is NonLeafNode =>
  isBaseNode(obj) && !('onAction' in obj) && ('options' in obj || 'onCustomInput' in obj);
export const isTreeNode = (obj: unknown): obj is TreeNode =>
  isNLNode(obj) || isLeafNode(obj);

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
export const traverseTree = async (
  address: string[],
  startNode: NonLeafNode,
): Promise<TreePath> => {
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
