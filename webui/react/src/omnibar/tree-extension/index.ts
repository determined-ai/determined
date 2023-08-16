import Fuse from 'fuse.js';

import root from 'omnibar/tree-extension/trees/index';
import { Children, LeafNode, NonLeafNode, TreeNode, TreePath } from 'omnibar/tree-extension/types';
import { getNodeChildren, isLeafNode, isNLNode, traverseTree } from 'omnibar/tree-extension/utils';
import { message } from 'utils/dialogApi';
import handleError, { ErrorType } from 'utils/error';
import { noOp } from 'utils/service';

const SEPARATOR = ' ';

interface TreeRequest {
  path: TreePath;
  query: string;
}

const parseInput = async (input: string, root: NonLeafNode): Promise<TreeRequest> => {
  const repeatedSeparator = new RegExp(SEPARATOR + '+', 'g');
  const cleanedInput = input.replace(repeatedSeparator, SEPARATOR);
  const sections = cleanedInput.split(SEPARATOR);
  const query = sections[sections.length - 1];
  const address = sections.slice(0, sections.length - 1);
  const path = await traverseTree(address, root);
  return {
    path,
    query,
  };
};

const absPathToAddress = (path: TreePath): string[] => path.map((tn) => tn.title).slice(1);

const noResultsNode: LeafNode = {
  closeBar: true,
  label: 'no matching options',
  onAction: noOp,
  title: 'Exit',
};

const queryTree = async (input: string, root: NonLeafNode): Promise<Children> => {
  const { path, query } = await parseInput(input, root);
  const node = path[path.length - 1];
  const children = await getNodeChildren(node);
  const fuse = new Fuse(children, {
    includeScore: false,
    keys: ['title', 'aliases', 'label'],
    minMatchCharLength: 1,
    shouldSort: true,
    threshold: 0.4,
  });
  const matches = query === '' ? children : fuse.search(query).map((r) => r.item);

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

// FIXME when querying async paths in a tree, while the extension is getting queried
// the previous options are still available and can be executed which means the user
// could see out of date options.
export const extension = async (input: string): Promise<Children> => {
  try {
    // query the default tree.
    return await queryTree(input, root);
  } catch (e) {
    handleError(e, {
      publicSubject: 'Failed to query omnibar.',
      type: ErrorType.Ui,
    });
    return [];
  }
};

export const onAction = async (
  inputEl: HTMLInputElement,
  item: TreeNode,
  query: (inputEl: string) => void,
): Promise<void> => {
  const { path } = await parseInput(inputEl.value, root);
  // update the omnibar text to reflect the current path
  inputEl.value =
    (path.length > 1 ? absPathToAddress(path).join(SEPARATOR) + SEPARATOR : '') + item.title;
  if (isLeafNode(item)) {
    await item.onAction(item);
    // if we opt to auto close the bar for user in some scenarios this
    // would be the place to check for it.
    message.info('Action executed.', 1);
  } else {
    // trigger the query.
    inputEl.value = inputEl.value + SEPARATOR;
    query(inputEl.value);
  }
};
