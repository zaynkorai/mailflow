import {
  _id,
  _times,
  _message,
  _fullName,
} from './_mock';

// ----------------------------------------------------------------------

export const _myAccount = {
  displayName: 'Zayn',
  email: 'zaynulabdin313@gmail.com',
  photoURL: '/assets/images/avatar/avatar-25.webp',
};

// ----------------------------------------------------------------------

export const _users = [...Array(24)].map((_, index) => ({
  id: _id(index),
  name: _fullName(index),
  message: _message(index),
  avatarUrl: `/assets/images/avatar/avatar-${index + 1}.webp`,
}));

// ----------------------------------------------------------------------

const COLORS = [
  '#00AB55',
  '#000000',
  '#FFFFFF',
  '#FFC0CB',
  '#FF4842',
  '#1890FF',
  '#94D82D',
  '#FFC107',
];

// ----------------------------------------------------------------------

export const _notifications = [
  {
    id: _id(1),
    title: 'Coming Soon',
    description: 'waiting for development',
    avatarUrl: null,
    type: 'order-placed',
    postedAt: _times(1),
    isUnRead: true,
  },
  {
    id: _id(2),
    title: 'Coming Soon',
    description: 'waiting for development',
    avatarUrl: null,
    type: 'chat-message',
    postedAt: _times(5),
    isUnRead: false,
  },
];
