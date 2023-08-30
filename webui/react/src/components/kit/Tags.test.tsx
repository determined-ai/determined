import { render, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import TagList, {
  ARIA_LABEL_CONTAINER,
  ARIA_LABEL_TRIGGER,
  tagsActionHelper,
} from 'components/kit/Tags';

const initTags = ['hello', 'world', 'space gap'].sort();

const setup = (tags: string[] = []) => {
  const handleOnChange = vi.fn();
  const view = render(<TagList tags={tags} onAction={tagsActionHelper(tags, handleOnChange)} />);
  const user = userEvent.setup();
  return { handleOnChange, user, view };
};

describe('TagList', () => {
  it('displays list of tags in order', () => {
    const { view } = setup(initTags);

    const container = view.getByLabelText(ARIA_LABEL_CONTAINER);
    initTags.forEach((tag, index) => {
      expect(view.getByText(tag)).toBeInTheDocument();
      expect(container.children[index].textContent).toBe(tag);
    });
  });

  it('handles tag addition', async () => {
    const addition = 'fox';
    const { handleOnChange, view, user } = setup();

    const trigger = view.getByLabelText(ARIA_LABEL_TRIGGER);
    await user.click(trigger);

    await user.keyboard(addition);
    await user.click(view.getByLabelText(ARIA_LABEL_CONTAINER));
    expect(handleOnChange).toHaveBeenCalledWith([addition]);
  });

  it('handles tag removal', async () => {
    const removalIndex = Math.floor(Math.random() * initTags.length);
    const removalTag = initTags[removalIndex];
    const resultTags = [...initTags.slice(0, removalIndex), ...initTags.slice(removalIndex + 1)];
    const { handleOnChange, view, user } = setup(initTags);

    const tag = view.getByText(removalTag).closest('[id]') as HTMLElement;
    expect(tag).not.toBeNull();

    const tagClose = within(tag).getByLabelText('close');
    await user.click(tagClose);

    expect(handleOnChange).toHaveBeenCalledWith(resultTags);
  });

  it('handles tag renaming', async () => {
    const rename = 'jump';
    const renameIndex = Math.floor(Math.random() * initTags.length);
    const renameTag = initTags[renameIndex];
    const { handleOnChange, user, view } = setup(initTags);

    const tag = view.getByText(renameTag);
    await user.click(tag);

    await user.keyboard(rename);
    await user.click(view.getByLabelText(ARIA_LABEL_CONTAINER));

    const updatedIndex = initTags.findIndex((tag: string) => tag === renameTag);
    initTags[updatedIndex] = rename;
    expect(handleOnChange).toHaveBeenCalledWith(initTags);
  });
});
