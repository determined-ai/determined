import { render, screen, waitFor } from '@testing-library/react';
import dayjs from 'dayjs';

import {
  DURATION_DAY,
  DURATION_HOUR,
  DURATION_MINUTE,
  DURATION_SECOND,
  DURATION_YEAR,
} from 'utils/datetime';

import TimeAgo, { TimeAgoCase } from './TimeAgo';

describe('TimeAgo', () => {
  const shared = { now: Date.now() };
  const offsetDays = 5 * DURATION_DAY;
  const daysMatch = /5d ago/i;

  beforeEach(() => {
    shared.now = Date.now();
  });

  it('should render with datetime as a string', () => {
    const datetimeString = new Date(shared.now - offsetDays).toISOString();
    render(<TimeAgo datetime={datetimeString} />);
    expect(screen.getByText(daysMatch)).toBeInTheDocument();
  });

  it('should render with datetime as a number', () => {
    const datetimeNumber = shared.now - offsetDays;
    render(<TimeAgo datetime={datetimeNumber} />);
    expect(screen.getByText(daysMatch)).toBeInTheDocument();
  });

  it('should render with datetime as a Date object', () => {
    const datetimeDate = new Date(shared.now - offsetDays);
    render(<TimeAgo datetime={datetimeDate} />);
    expect(screen.getByText(daysMatch)).toBeInTheDocument();
  });

  it('should render with datetime as a Dayjs object', () => {
    const datetimeDayjs = dayjs(shared.now - offsetDays);
    render(<TimeAgo datetime={datetimeDayjs} />);
    expect(screen.getByText(daysMatch)).toBeInTheDocument();
  });

  it('should render with custom className', () => {
    const datetime = shared.now - offsetDays;
    const customClassName = 'customClassName';
    const view = render(<TimeAgo className={customClassName} datetime={datetime} />);
    const element = view.getByText(daysMatch);
    expect(element.className).toBe(`base ${customClassName}`);
  });

  it('should render "Just Now" when < 1 minute', () => {
    render(<TimeAgo datetime={shared.now - DURATION_SECOND} />);
    expect(screen.getByText(/just now/i)).toBeInTheDocument();
  });

  it('should render a minute', () => {
    render(<TimeAgo datetime={shared.now - DURATION_MINUTE} />);
    expect(screen.getByText(/1m ago/i)).toBeInTheDocument();
  });

  it('should render an hour', () => {
    render(<TimeAgo datetime={shared.now - DURATION_HOUR} />);
    expect(screen.getByText(/1h ago/i)).toBeInTheDocument();
  });

  it('should render a day', () => {
    render(<TimeAgo datetime={shared.now - DURATION_DAY} />);
    expect(screen.getByText(/1d ago/i)).toBeInTheDocument();
  });

  it('should render a week', () => {
    render(<TimeAgo datetime={shared.now - 7 * DURATION_DAY} />);
    expect(screen.getByText(/1w ago/i)).toBeInTheDocument();
  });

  it('should render a month', () => {
    render(<TimeAgo datetime={shared.now - 31 * DURATION_DAY} />);
    expect(screen.getByText(/1mo ago/i)).toBeInTheDocument();
  });

  it('should render date when > 1 year', () => {
    render(<TimeAgo datetime={shared.now - DURATION_YEAR} />);
    expect(screen.getByText(/\w{3} \d{1,2}, \d{4}/i)).toBeInTheDocument();
  });

  it('should render multiple units', () => {
    const datetime = shared.now - DURATION_DAY - DURATION_HOUR - DURATION_MINUTE;
    render(<TimeAgo datetime={datetime} units={3} />);
    expect(screen.getByText(/1d 1h 1m ago/i)).toBeInTheDocument();
  });

  it('should render with custom date format when > 1 year', () => {
    const datetime = shared.now - DURATION_YEAR;
    const format = 'YYYY MMM DD';
    render(<TimeAgo dateFormat={format} datetime={datetime} />);
    expect(screen.getByText(/\d{4} \w{3} \d{2}/i)).toBeInTheDocument();
  });

  it('should render long format', () => {
    render(<TimeAgo datetime={shared.now - DURATION_DAY} long />);
    expect(screen.getByText(/1 day ago/i)).toBeInTheDocument();
  });

  it('should render plural in long format', () => {
    render(<TimeAgo datetime={shared.now - offsetDays} long />);
    expect(screen.getByText(/5 days ago/i)).toBeInTheDocument();
  });

  it('should render multiple units in long format', () => {
    const datetime = shared.now - DURATION_DAY - DURATION_HOUR - DURATION_MINUTE;
    render(<TimeAgo datetime={datetime} long units={3} />);
    expect(screen.getByText(/1 day 1 hour 1 minute ago/i)).toBeInTheDocument();
  });

  it('should render updates', async () => {
    render(<TimeAgo datetime={shared.now - 59 * DURATION_SECOND} />);
    expect(screen.getByText(/just now/i)).toBeInTheDocument();
    await new Promise((r) => setTimeout(r, 2000));
    await waitFor(() => expect(screen.queryByText(/1m ago/i)).not.toBeNull());
    expect(screen.getByText(/1m ago/i)).toBeInTheDocument();
  });

  it('should not render updates', async () => {
    vi.useFakeTimers();

    render(<TimeAgo datetime={shared.now - 59 * DURATION_SECOND} noUpdate />);
    expect(screen.getByText(/just now/i)).toBeInTheDocument();

    await vi.advanceTimersByTime(2000);
    expect(screen.getByText(/just now/i)).toBeInTheDocument();

    vi.useRealTimers();
  });

  it('should render lower case', () => {
    render(<TimeAgo datetime={shared.now - DURATION_SECOND} stringCase={TimeAgoCase.Lower} />);
    expect(screen.getByText(/just now/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} stringCase={TimeAgoCase.Lower} />);
    expect(screen.getByText(/5d ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} long stringCase={TimeAgoCase.Lower} />);
    expect(screen.getByText(/5 days ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - DURATION_YEAR} stringCase={TimeAgoCase.Lower} />);
    expect(screen.getByText(/[a-z]{3} \d{1,2}, \d{4}/)).toBeInTheDocument();
  });

  it('should render sentence case', () => {
    render(<TimeAgo datetime={shared.now - DURATION_SECOND} stringCase={TimeAgoCase.Sentence} />);
    expect(screen.getByText(/Just now/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} stringCase={TimeAgoCase.Sentence} />);
    expect(screen.getByText(/5d ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} long stringCase={TimeAgoCase.Sentence} />);
    expect(screen.getByText(/5 days ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - DURATION_YEAR} stringCase={TimeAgoCase.Sentence} />);
    expect(screen.getByText(/[A-Z][a-z]{2} \d{1,2}, \d{4}/)).toBeInTheDocument();
  });

  it('should render title case', () => {
    render(<TimeAgo datetime={shared.now - DURATION_SECOND} stringCase={TimeAgoCase.Title} />);
    expect(screen.getByText(/Just Now/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} stringCase={TimeAgoCase.Title} />);
    expect(screen.getByText(/5d Ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - offsetDays} long stringCase={TimeAgoCase.Title} />);
    expect(screen.getByText(/5 Days Ago/)).toBeInTheDocument();

    render(<TimeAgo datetime={shared.now - DURATION_YEAR} stringCase={TimeAgoCase.Title} />);
    expect(screen.getByText(/[A-Z][a-z]{2} \d{1,2}, \d{4}/)).toBeInTheDocument();
  });
});
