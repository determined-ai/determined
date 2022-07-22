import { render, screen } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';

import { MessageType } from './Message';
import Message, { Props } from './Message';

const setup = (props: Props) => {
  const handleOnChange = jest.fn();
  const view = render(
    <StoreProvider>
      <Message {...props} />
    </StoreProvider>,
  );
  return { handleOnChange, view };
};

describe('Message', () => {
  it('should display Alert title, message and image', () => {
    setup({ message: 'Alert message', title: 'Alert title', type: MessageType.Alert });
    expect(screen.getByText('Alert message')).toBeInTheDocument();
    expect(screen.getByText('Alert title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Alert, { exact: false })).toBeInTheDocument();
  });

  it('should display Warning title, message and image', () => {
    setup({ message: 'Warning message', title: 'Warning title', type: MessageType.Warning });
    expect(screen.getByText('Warning message')).toBeInTheDocument();
    expect(screen.getByText('Warning title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Warning, { exact: false })).toBeInTheDocument();
  });

  it('should display Empty title, message and image', () => {
    setup({ message: 'Empty message', title: 'Empty title', type: MessageType.Empty });
    expect(screen.getByText('Empty message')).toBeInTheDocument();
    expect(screen.getByText('Empty title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Empty, { exact: false })).toBeInTheDocument();
  });
});
