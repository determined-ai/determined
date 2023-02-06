import crypto from 'crypto';

import type { Plugin } from 'vite';

// incomplete list of directives
type CspHashDirective = 'script-src' | 'style-src';
type CspDirective = 'base-uri' | 'frame-src' | 'object-src' | CspHashDirective;

type CspRuleConfig = {
  [key in CspDirective]?: string[];
};

type CspHashConfig = {
  [key in CspHashDirective]?: boolean;
};

interface CspHtmlPluginConfig {
  cspRules: CspRuleConfig;
  hashEnabled: CspHashConfig;
}

export const cspHtml = ({ cspRules, hashEnabled = {} }: CspHtmlPluginConfig): Plugin => ({
  name: 'csp-html',
  transformIndexHtml: {
    async handler(html: string) {
      const finalCspRules: CspRuleConfig = {
        'base-uri': ["'self'"],
        ...cspRules,
      };
      const hashRules = Object.entries(hashEnabled) as [CspHashDirective, boolean][];
      if (hashRules.length) {
        const cheerio = await import('cheerio');
        const $ = cheerio.load(html);
        hashRules.forEach(([directive, enabled]: [CspHashDirective, boolean]) => {
          if (!enabled) return;
          const [tag] = directive.split('-');
          $(tag).each((_, el) => {
            const source = $(el).html();
            if (source) {
              const hash = crypto.createHash('sha256').update(source).digest('base64');
              finalCspRules[directive] = (finalCspRules[directive] || []).concat([
                `'sha256-${hash}'`,
              ]);
            }
          });
        });
      }
      const content = Object.entries(finalCspRules)
        .map(([directive, sources]) => `${directive} ${sources.join(' ')}`)
        .join('; ');
      return [
        {
          attrs: {
            content,
            'http-equiv': 'Content-Security-Policy',
          },
          tag: 'meta',
        },
      ];
    },
    order: 'post',
  },
});
