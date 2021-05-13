import { message } from 'antd';
import Fuse from 'fuse.js';

import { StoreAction } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { store } from 'omnibar/exposedStore';
import root from 'omnibar/tree-extension/trees/index';
import {
  BaseNode, Children, LeafNode, NonLeafNode, TreePath,
} from 'omnibar/tree-extension/types';
import { getNodeChildren, isLeafNode,
  isNLNode, traverseTree } from 'omnibar/tree-extension/utils';
import { noOp } from 'services/utils';

const SEPARATOR = ' ';

interface TreeRequest {
  path: TreePath;
  query: string;
}

const parseInput = async (input: string, root: NonLeafNode): Promise<TreeRequest> => {
  const repeatedSeparator = new RegExp(SEPARATOR + '+', 'g');
  const cleanedInput = input.replace(repeatedSeparator, SEPARATOR);
  const sections = cleanedInput.split(SEPARATOR);
  const query = sections[sections.length-1];
  const address = sections.slice(0,sections.length-1);
  const path = await traverseTree(address, root);
  return {
    path,
    query,
  };
};

const absPathToAddress = (path: TreePath): string[] => (path.map(tn => tn.title).slice(1));

const noResultsNode: LeafNode = {
  closeBar: true,
  label: 'no matching options',
  onAction: noOp,
  title: 'Exit',
};

const queryTree = async (input: string, root: NonLeafNode): Promise<Children> => {
  const { path, query } = await parseInput(input, root);
  const node = path[path.length-1];
  const children = await getNodeChildren(node);
  const fuse = new Fuse(
    children,
    {
      includeScore: false,
      keys: [ 'title', 'aliases', 'label' ],
      minMatchCharLength: 2,
      shouldSort: true,
      threshold: 0.4,
    },
  );
  const matches = query === '' ? children : fuse.search(query).map(r => r.item);

  if (isNLNode(node)) {
    if (node.onCustomInput) {
      const moreOptions = await node.onCustomInput(query);
      matches.push(...moreOptions);
    } else if (matches.length === 0) {
      matches.push(noResultsNode);
    }
  }
  return matches;
};

export const extension = async(input: string): Promise<Children> => {
  try {
    // query the default tree.
    return await queryTree(input, root);
  } catch (e) {
    handleError({
      error: e,
      message: 'failed to query omnibar',
      type: ErrorType.Ui,
    });
    return [];
  }
};

export const onAction = async (
  item: BaseNode,
  query: (input: string) => void,
): Promise<void> => {
  const input: HTMLInputElement|null = document.querySelector('#omnibar input[type="text"]');
  if (!input) return Promise.resolve();
  const { path } = await parseInput(input.value, root);
  // update the omnibar text to reflect the current path
  input.value = (path.length > 1 ? absPathToAddress(path).join(SEPARATOR) + SEPARATOR : '')
        + item.title;
  if (isLeafNode(item)) {
    await item.onAction(item);
    if (item.closeBar || path.find(n => n.title === 'goto')) {
      store?.dispatch({ type: StoreAction.HideOmnibar });
    } else {
      message.info('Action executed.', 1);
    }
  } else {
    // trigger the query.
    input.value = input.value + SEPARATOR;
    query(input.value);
  }
  return Promise.resolve();
};
