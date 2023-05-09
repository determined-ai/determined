import { render, screen } from '@testing-library/react';
import { CSSProperties } from 'react';

import { LogLevel } from 'types';

import LogViewerEntry, { Props } from './LogViewerEntry';

const setup = (props: Props) => {
  return render(<LogViewerEntry {...props} />);
};

describe('LogViewerEntry', () => {
  const formattedTime = '[2022-02-22 21:24:37]';
  const level = LogLevel.Error;
  const message = 'Uh-oh there is a boo-boo.';

  it('should render log entry', async () => {
    setup({ formattedTime, level, message });
    expect(await screen.getByText(formattedTime)).toBeInTheDocument();
    expect(await screen.getByText(message)).toBeInTheDocument();
  });

  it('should render with all level types except None', () => {
    Object.values(LogLevel).forEach((level) => {
      if (level === LogLevel.None) return;
      const { container } = setup({ formattedTime, level: level, message });
      const icon = container.querySelector(`.icon-${level}`);
      expect(icon).not.toBeNull();
      expect(icon).toBeInTheDocument();
    });
  });

  it('should render without wrapping', () => {
    const { container } = setup({ formattedTime, level, message, noWrap: true });
    const noWrapLogEntry = container.querySelector('[class*="noWrap"]');
    expect(noWrapLogEntry).not.toBeNull();
    expect(noWrapLogEntry).toBeInTheDocument();
  });

  it('should apply custom time style', () => {
    const timeStyle: CSSProperties = { backgroundColor: 'HotPink' };
    const { container } = setup({ formattedTime, level, message, timeStyle });
    const hotPinkLogEntry = container.querySelector('[style*="HotPink"]');
    expect(hotPinkLogEntry).not.toBeNull();
    expect(hotPinkLogEntry).toBeInTheDocument();
  });
});
