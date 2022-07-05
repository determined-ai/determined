import { render, screen } from '@testing-library/react';
import React from 'react';

import { MessageType } from './Message';
import Message, { Props } from './Message';

const setup = (props: Props) => {
  const handleOnChange = jest.fn();
  const view = render(<Message {...props} />);
  return { handleOnChange, view };
};

describe('Message', () => {

  it('Alert message displays title, message, image', () => {
    setup({ message: 'Alert message', title: 'Alert title', type: MessageType.Alert });
    expect(screen.getByText('Alert message')).toBeInTheDocument();
    expect(screen.getByText('Alert title')).toBeInTheDocument();
    expect(screen.getByAltText(MessageType.Alert)).toBeInTheDocument();
  });

  it('Warning message displays title, message, image', () => {
    setup({ message: 'Warning message', title: 'Warning title', type: MessageType.Warning });
    expect(screen.getByText('Warning message')).toBeInTheDocument();
    expect(screen.getByText('Warning title')).toBeInTheDocument();
    expect(screen.getByAltText(MessageType.Warning)).toBeInTheDocument();
  });

  it('Empty message displays title, message, image', () => {
    setup({ message: 'Empty message', title: 'Empty title', type: MessageType.Empty });
    expect(screen.getByText('Empty message')).toBeInTheDocument();
    expect(screen.getByText('Empty title')).toBeInTheDocument();
  });

});
