import css from './Link.module.scss';

interface Props {
  children?: React.ReactNode;
  onClick?: React.MouseEventHandler<HTMLAnchorElement>;
  href?: string;
  rel?: string;
  disabled?: boolean;
  size?: 'tiny' | 'small' | 'medium' | 'large';
}

const Link: React.FC<Props> = ({ size, onClick, href, rel, disabled, ...props }: Props) => {
  const classes = [css.base];
  if (disabled) classes.push(css.disabled);
  if (size) classes.push(css[size]);

  if (disabled) {
    return <span className={classes.join(' ')}>{props.children}</span>;
  }

  return (
    <a aria-label={href} className={classes.join(' ')} href={href} rel={rel} onClick={onClick}>
      {props.children}
    </a>
  );
};

export default Link;
