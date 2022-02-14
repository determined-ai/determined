import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import TagList, { ARIA_LABEL_CONTAINER, ARIA_LABEL_INPUT, ARIA_LABEL_TRIGGER } from './TagList';

const initTags = [ 'hello', 'world', 'space gap' ].sort();

const setup = (tags: string[] = []) => {
  const handleOnChange = jest.fn();
  const view = render(<TagList tags={tags} onChange={handleOnChange} />);
  return { handleOnChange, view };
};

describe('TagList', () => {
  it('displays list of tags in order', () => {
    setup(initTags);

    const container = screen.getByLabelText(ARIA_LABEL_CONTAINER);
    initTags.forEach((tag, index) => {
      expect(screen.getByText(tag)).toBeInTheDocument();
      expect(container.children[index].textContent).toBe(tag);
    });
  });

  it('handles tag addition', () => {
    const addition = 'fox';
    const { handleOnChange } = setup();

    const trigger = screen.getByLabelText(ARIA_LABEL_TRIGGER);
    userEvent.click(trigger);

    const input = screen.getByRole('textbox', { name: ARIA_LABEL_INPUT });
    userEvent.type(input, `${addition}{enter}`);

    expect(handleOnChange).toHaveBeenCalledWith([ addition ]);
  });

  it('handles tag removal', () => {
    const removalIndex = Math.floor(Math.random() * initTags.length);
    const removalTag = initTags[removalIndex];
    const resultTags = [ ...initTags.slice(0, removalIndex), ...initTags.slice(removalIndex + 1) ];
    const { handleOnChange } = setup(initTags);

    const tag = screen.getByText(removalTag).closest('[id]') as HTMLElement;
    expect(tag).not.toBeNull();

    const tagClose = within(tag).getByLabelText('close');
    userEvent.click(tagClose);

    expect(handleOnChange).toHaveBeenCalledWith(resultTags);
  });

  it('handles tag renaming', () => {
    const rename = 'jump';
    const renameIndex = Math.floor(Math.random() * initTags.length);
    const renameTag = initTags[renameIndex];
    const { handleOnChange } = setup(initTags);

    const tag = screen.getByText(renameTag);
    userEvent.click(tag);

    const input = screen.getByLabelText(ARIA_LABEL_INPUT);
    userEvent.type(input, `${rename}{enter}`);

    //screen.debug();

    const resultTags = initTags.filter((tag: string) => tag !== renameTag);
    resultTags.push(rename);

    expect(handleOnChange).toHaveBeenCalledWith(resultTags);
  });
});
