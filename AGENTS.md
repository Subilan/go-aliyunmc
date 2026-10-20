# go-aliyunmc Agent Development Principle

Hey Agent. This file documents your principle of development. It's pretty normal not knowing what's the best practice. There it is.

## Comment Style

Always comment in Chinese and Go doc style. Comment what the subject does clearly with minimal tokens. DO NOT include preamables, prose, why & why-not, considerations, user guidance, etc.

Code is code, not essay or knowledge base.

## Fallback Logic, Compatibility Layer and Considerations

Always consider the WIP non-production unless told.

Less logic, less compatibility layer, the better. Add compatibility layer only when there's actual need. If you are not sure, just ask. Do not add such thing without confirmation, as it would make things A WHOLE LOT messy.

## No CLAUDE.md, Read yourself

You should explore the codebase yourself to gain information for your task. There's no CLAUDE.md and won't be, because it's a place that can be easily filled by expired information and random details, rules and considerations, interwoven.

## Use BOLD format to give points