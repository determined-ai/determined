import { Children, getNodeChildren, isLeafNode,
  isTreeNode, traverseTree, TreePath } from 'AsyncTree';
import root from 'omnibar/sampleTree';

const SEPARATOR = ' ';

interface TreeRequest {
  query: string;
  path: TreePath;
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

const absPathToAddress = (path: TreePath): string[] =>  (path.map(tn => tn.title).slice(1));

const query = async (input: string): Promise<Children> => {
  const { path, query } = await parseInput(input);
  const node = path[path.length-1];
  if (isLeafNode(node)) {
    // this is after the leafnode onaction triggers
    // could do an execute confirmation.
    return [];
  }
  const children = await getNodeChildren(node);

  const matches = children.filter(it => it.title.match(new RegExp(query, 'i')));
  return matches;
};

export const extension = async(input: string): Promise<Children> => {
  try {
    return await query(input);
  } catch (e) {
    console.error(e);
    // omnibar eatsup the exceptions
    // throw e;
    return [];
  }
};

export const onAction = async <T>(item: T): Promise<void> => {
  if (!!item && isTreeNode(item)) {
    // TODO should be replaced, perhaps, with a update to the omnibar package's command decorator
    // TODO setup the omnibar with context and tree
    // TODO make below a generic omnibar decorator
    const input: HTMLInputElement|null = document.querySelector('#omnibar input[type="text"]');
    if (input) {
      const { path } = await parseInput(input.value);
      input.value = (path.length > 1 ?  absPathToAddress(path).join(SEPARATOR) + SEPARATOR  : '')
        + item.title;
      // trigger the onchange
      input.onchange && input.onchange(undefined as unknown as Event);
    }
    if (isLeafNode(item)) return item.onAction(item);
  }
  // else meh
  return Promise.resolve();
};
