import { AdminDashboardData, DashboardData } from './types';

export const fallbackData: DashboardData = {
    channel: {
        id: -1002676384505,
        title: "Teste",
        newPackCaption: "╔═━──━═༻✧༺═━──━═╗\n\n        𖦹 ࣪ ⭑ ᥫ᭡\n        (｡•́︿•̀｡)っ✧.*ೃ༄\n        ˗ˏˋ [$title]($link) ⋆｡˚ ☁︎\n            彡♡ ₊˚\n\n⋆｡˚ ❀ @FreddyCaptionBot ☽⁺₊\n\n╚═━──━═༻✧༺═━──━═╝",
        newPackMessageButtons: true,
        newPackStickerButtons: true,
        newPackMessagePosition: 'above',
        newPackReplyToSticker: false,
        inviteUrl: "https://t.me/+0evIoTBoXmw3NzQx",
        ownerId: 7595607953,
        reactions: "👍,❤️,🔥,👏,🤔",
        reactionPosition: 2,
        dynamicLinks: true,
        dlBotButtons: true,
        dlBotCaptions: true,
        dlBotReactions: true,
        nativeReactionsEnabled: false,
        nativeReactions: "👍,❤️",
        nativeReactionMode: 'fixed',
        defaultCaption: {
            captionId: "fc99267a-57a5-4875-9f90-11567b2cb976",
            caption: "➽ 𝐛𝐲 @FreddyCaptionBot",
            messagePermission: {
                messagePermissionId: "9e23c5b1-bc13-4553-bc27-ab858ad98a9b",
                linkPreview: true, message: true, audio: true, video: true,
                photo: true, document: true, sticker: true, gif: true, reactions: true,
                ownerCaptionId: "fc99267a-57a5-4875-9f90-11567b2cb976",
                created_at: "2026-02-27T23:11:42.611695-03:00",
                updated_at: "2026-02-27T23:11:42.611695-03:00",
            },
            buttonsPermission: {
                buttonsPermissionId: "74d62e2d-a63f-4a92-915b-f522b1eca924",
                linkPreview: true, message: true, audio: true, video: true, photo: true,
                document: true, sticker: true, gif: true, reactions: true,
                ownerCaptionId: "fc99267a-57a5-4875-9f90-11567b2cb976",
                created_at: "2026-02-27T23:11:42.613143-03:00",
                updated_at: "2026-02-27T23:11:42.613143-03:00",
            },
            ownerChannelId: -1002676384505,
            created_at: "2026-02-27T23:11:42.610242-03:00",
            updated_at: "2026-02-27T23:11:42.610242-03:00",
        },
        buttons: [
            {
                buttonId: "b049f95f-55d1-4623-b55c-b3fdfdda6773",
                nameButton: "Teste", buttonUrl: "https://t.me/+0evIoTBoXmw3NzQx",
                positionX: 0, positionY: 0, ownerChannelId: -1002676384505,
                created_at: "2026-02-27T23:11:42.614454-03:00",
                updated_at: "2026-02-27T23:11:42.614454-03:00",
            },
            {
                buttonId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
                nameButton: "Canal", buttonUrl: "https://t.me/+ABC123def456",
                positionX: 1, positionY: 0, ownerChannelId: -1002676384505,
                created_at: "2026-02-27T23:11:42.614454-03:00",
                updated_at: "2026-02-27T23:11:42.614454-03:00",
            },
            {
                buttonId: "f7e8d9c0-b1a2-3456-7890-abcdef123456",
                nameButton: "Suporte", buttonUrl: "https://t.me/+XYZ789ghi012",
                positionX: 0, positionY: 1, ownerChannelId: -1002676384505,
                created_at: "2026-02-27T23:11:42.614454-03:00",
                updated_at: "2026-02-27T23:11:42.614454-03:00",
            },
        ],
        customCaptions: [],
        created_at: "2026-02-27T23:11:42.608678-03:00",
        updated_at: "2026-02-27T23:11:42.608678-03:00",
    },
    user: {
        id: 7595607953,
        first_name: ".",
        username: "@admin",
        is_admin: true,
        is_blacklisted: false,
        isContribute: false,
        channels: null,
        created_at: "2026-02-27T22:54:15.504001-03:00",
        updated_at: "2026-02-27T23:11:42.076272-03:00",
    },
};

export const mockAdminData: AdminDashboardData = {
    success: true,
    users: [
        {
            id: 7595607953,
            first_name: "Admin User",
            username: "@freddy_admin",
            is_admin: true,
            is_blacklisted: false,
            isContribute: true,
            created_at: "2026-08-01T22:54:15.504001-03:00",
            updated_at: "2026-08-02T23:11:42.076272-03:00",
            channels: [fallbackData.channel]
        },
        {
            id: 12345678,
            first_name: "Regular User",
            username: "@regular",
            is_admin: false,
            is_blacklisted: false,
            isContribute: false,
            created_at: "2026-08-02T10:00:00.000000-03:00",
            updated_at: "2026-08-02T10:00:00.000000-03:00",
            channels: []
        },
        {
            id: 98765432,
            first_name: "Maria Silva",
            username: "@mariasilva",
            is_admin: false,
            is_blacklisted: false,
            isContribute: true,
            created_at: "2026-07-05T14:20:00.000000-03:00",
            updated_at: "2026-08-01T14:20:00.000000-03:00",
            channels: [
                { ...fallbackData.channel, id: -100999888777, title: "Canal de Receitas" },
                { ...fallbackData.channel, id: -100555444333, title: "Dicas de Python" }
            ]
        },
        {
            id: 55443322,
            first_name: "João Tech",
            username: "@joaotech",
            is_admin: false,
            is_blacklisted: false,
            isContribute: false,
            created_at: "2026-06-10T09:00:00.000000-03:00",
            updated_at: "2026-07-10T09:00:00.000000-03:00",
            channels: []
        },
        {
            id: 11223344,
            first_name: "Lucas Games",
            username: "@lucasgames",
            is_admin: false,
            is_blacklisted: true,
            isContribute: true,
            created_at: "2026-05-12T18:30:00.000000-03:00",
            updated_at: "2026-07-30T18:30:00.000000-03:00",
            channels: Array(5).fill(0).map((_, i) => ({ ...fallbackData.channel, id: -1000000000 + i, title: `Game Stream ${i + 1}` }))
        }
    ],
    channels: [
        fallbackData.channel,
        { ...fallbackData.channel, id: -100123456789, title: "Outro Canal" },
        { ...fallbackData.channel, id: -100999888777, title: "Canal de Receitas" },
        { ...fallbackData.channel, id: -100555444333, title: "Dicas de Python" },
        { ...fallbackData.channel, id: -100111222333, title: "Gadgets Review" },
        { ...fallbackData.channel, id: -100777666555, title: "Notícias Urgentes" },
        { ...fallbackData.channel, id: -100444333222, title: "Filmes e Séries" }
    ]
};
