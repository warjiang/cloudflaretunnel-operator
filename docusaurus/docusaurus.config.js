// @ts-check

const config = {
  title: 'Cloudflare Tunnel Operator',
  tagline: 'Kubernetes Operator for Cloudflare Tunnel',
  favicon: 'img/logo.svg',

  url: 'https://warjiang.github.io',
  baseUrl: '/cloudflaretunnel-operator/',

  organizationName: 'warjiang',
  projectName: 'cloudflaretunnel-operator',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'zh-CN'],
    localeConfigs: {
      en: {label: 'English'},
      'zh-CN': {label: '中文'}
    }
  },

  presets: [
    [
      'classic',
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js')
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css')
        }
      }
    ]
  ],

  themeConfig: {
    image: 'img/logo.svg',
    navbar: {
      title: 'Cloudflare Tunnel Operator',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'mainSidebar',
          position: 'left',
          label: 'Docs'
        },
        {
          type: 'localeDropdown',
          position: 'right'
        },
        {
          href: 'https://github.com/warjiang/cloudflaretunnel-operator',
          label: 'GitHub',
          position: 'right'
        }
      ]
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [{label: 'Introduction', to: '/'}]
        },
        {
          title: 'Community',
          items: [{label: 'GitHub', href: 'https://github.com/warjiang/cloudflaretunnel-operator'}]
        }
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Cloudflare Tunnel Operator contributors.`
    },
    prism: {
      additionalLanguages: ['bash', 'yaml']
    }
  }
};

module.exports = config;
