import Fuse from 'fuse.js';

import { StoreAction } from 'contexts/Store';
import { BaseNode, Children, getNodeChildren, isLeafNode,
  isNLNode, isTreeNode, traverseTree, TreePath } from 'omnibar/AsyncTree';
import { store } from 'omnibar/exposedStore';
import root from 'omnibar/sampleTree';

const SEPARATOR = ' ';

interface TreeRequest {
  path: TreePath;
  query: string;
}

const parseInput = async (input: string): Promise<TreeRequest> => {
  const sections = input.split(SEPARATOR);
  const query = sections[sections.length-1];
  const address = sections.slice(0,sections.length-1);
  const path = await traverseTree(address, root);
  return {
    path,
    query,
  };
};

const absPathToAddress = (path: TreePath): string[] => (path.map(tn => tn.title).slice(1));

const query = async (input: string): Promise<Children> => {
  const { path, query } = await parseInput(input);
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

  if (isNLNode(node) && node.onCustomInput) {
    // TODO provide hint if query is empty.
    const moreOptions = await node.onCustomInput(query);
    matches.push(...moreOptions);
  }
  return matches;
};

export const extension = async(input: string): Promise<Children> => {
  console.debug('extension triggered');
  try {
    return await query(input);
  } catch (e) {
    console.error(e);
    return [];
  }
};

export const onAction = async (item: BaseNode): Promise<void> => {
  console.debug('onaction called', item);
  if (!!item && isTreeNode(item)) {
    // TODO should be replaced, perhaps, with a update to the omnibar package's command decorator
    // TODO setup the omnibar with context and tree
    // TODO make below a generic omnibar decorator
    const input: HTMLInputElement|null = document.querySelector('#omnibar input[type="text"]');
    if (!input) return Promise.resolve();
    const { path } = await parseInput(input.value);
    // update the omnibar text to reflect the current path
    input.value = (path.length > 1 ? absPathToAddress(path).join(SEPARATOR) + SEPARATOR : '')
        + item.title; // TODO add the separator and trigger the extensions
    // trigger the onchange
    input.onchange && input.onchange(undefined as unknown as Event);
    if (isLeafNode(item)) {
      const actionRv = await item.onAction(item);
      if (item.closeBar) store?.dispatch({ type: StoreAction.HideOmnibar });
      // always close the bar for location modules.
      if (path.find(n => n.title === 'goto')) {
        store?.dispatch({ type: StoreAction.HideOmnibar });
      }
      return actionRv;
    }
  }
  // else meh
  return Promise.resolve();
};
