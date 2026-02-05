import 'vuetify/lib/styles/main.css'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'

export default createVuetify({
  theme: {
    defaultTheme: 'dark',
    themes: {
      dark: {
        colors: {
          primary: '#FFC107',
          secondary: '#FF9800',
          accent: '#FFD54F',
          error: '#FF5252',
          success: '#4CAF50',
          warning: '#FB8C00',
          info: '#2196F3',
          surface: '#1E1E1E',
          background: '#121212',
        },
      },
    },
  },
})
