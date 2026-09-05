<template>
  <time :datetime="val" :title="title">{{ formatted }}</time>
</template>

<script lang="ts">
import { dateRepr, dateTimeString } from "../utils";
import { defineComponent } from "vue";

export default defineComponent({
  props: ["val", "locale"],
  data() {
    return {
      date: new Date(this.val),
      formatted: "" as string,
      interval: undefined as number | undefined,
    };
  },
  computed: {
    title(): string {
      return dateTimeString(this.date, this.locale);
    },
  },
  created() {
    this.repaint();
  },
  mounted() {
    this.interval = setInterval(() => {
      this.repaint();
    }, 600000); // every 10 minutes
  },
  methods: {
    repaint() {
      this.formatted = dateRepr(this.date, this.locale);
    },
  },
  watch: {
    locale() {
      this.repaint();
    },
  },
  unmounted() {
    clearInterval(this.interval);
  },
});
</script>
