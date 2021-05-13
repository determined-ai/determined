import { isAsyncFunction } from 'utils/data';

import { Children, LeafNode, NLNode, TreeNode, TreePath } from './types';

export const isLeafNode = (node: TreeNode): node is LeafNode =>
  'onAction' in node && !('options' in node);
export const isNLNode = (node: TreeNode): node is NLNode =>
  !('onAction' in node) && ('options' in node || 'onCustomInput' in node);
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
