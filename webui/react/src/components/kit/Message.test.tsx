import { render, screen } from '@testing-library/react';

import { UIProvider } from 'components/kit/Theme';

import Message, { MessageType, Props } from './Message';

const setup = (props: Props) => {
  const handleOnChange = vi.fn();
  const view = render(
    <UIProvider>
      <Message {...props} />
    </UIProvider>,
  );
  return { handleOnChange, view };
};

describe('Message', () => {
  it('should display Error title, message and image', () => {
    setup({ body: 'Error message', title: 'Error title', type: MessageType.Error });
    expect(screen.getByText('Error message')).toBeInTheDocument();
    expect(screen.getByText('Error title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Error, { exact: false })).toBeInTheDocument();
  });

  it('should display Warning title, message and image', () => {
    setup({ body: 'Warning message', title: 'Warning title', type: MessageType.Warning });
    expect(screen.getByText('Warning message')).toBeInTheDocument();
    expect(screen.getByText('Warning title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Warning, { exact: false })).toBeInTheDocument();
  });

  it('should display Info title, message and image', () => {
    setup({ body: 'Info message', title: 'Info title', type: MessageType.Info });
    expect(screen.getByText('Info message')).toBeInTheDocument();
    expect(screen.getByText('Info title')).toBeInTheDocument();
    expect(screen.getByTitle(MessageType.Info, { exact: false })).toBeInTheDocument();
  });
});
